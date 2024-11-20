// Copyright 2020 ConsenSys Software Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by gnark DO NOT EDIT

package plonk

import (
	"fmt"
	"github.com/consensys/gnark-crypto/ecc/bls12-377/fr"
	"github.com/consensys/gnark-crypto/ecc/bls12-377/fr/fft"
	"github.com/consensys/gnark-crypto/ecc/bls12-377/fr/iop"
	"github.com/consensys/gnark-crypto/ecc/bls12-377/kzg"
	"github.com/BaoNinh2808/gnark/backend/plonk/internal"
	"github.com/consensys/gnark/constraint"
	cs "github.com/consensys/gnark/constraint/bls12-377"
)

// VerifyingKey stores the data needed to verify a proof:
// * The commitment scheme
// * Commitments of ql prepended with as many ones as there are public inputs
// * Commitments of qr, qm, qo, qk prepended with as many zeroes as there are public inputs
// * Commitments to S1, S2, S3
type VerifyingKey struct {
	// Size circuit
	Size              uint64
	SizeInv           fr.Element
	Generator         fr.Element
	NbPublicVariables uint64

	// Commitment scheme that is used for an instantiation of PLONK
	Kzg kzg.VerifyingKey

	// cosetShift generator of the coset on the small domain
	CosetShift fr.Element

	// S commitments to S1, S2, S3
	S [3]kzg.Digest

	// Commitments to ql, qr, qm, qo, qcp prepended with as many zeroes (ones for l) as there are public inputs.
	// In particular Qk is not complete.
	Ql, Qr, Qm, Qo, Qk kzg.Digest
	Qcp                []kzg.Digest

	CommitmentConstraintIndexes []uint64
}

// Trace stores a plonk trace as columns
type Trace struct {
	// Constants describing a plonk circuit. The first entries
	// of LQk (whose index correspond to the public inputs) are set to 0, and are to be
	// completed by the prover. At those indices i (so from 0 to nb_public_variables), LQl[i]=-1
	// so the first nb_public_variables constraints look like this:
	// -1*Wire[i] + 0* + 0 . It is zero when the constant coefficient is replaced by Wire[i].
	Ql, Qr, Qm, Qo, Qk *iop.Polynomial
	Qcp                []*iop.Polynomial

	// Polynomials representing the splitted permutation. The full permutation's support is 3*N where N=nb wires.
	// The set of interpolation is <g> of size N, so to represent the permutation S we let S acts on the
	// set A=(<g>, u*<g>, u^{2}*<g>) of size 3*N, where u is outside <g> (its use is to shift the set <g>).
	// We obtain a permutation of A, A'. We split A' in 3 (A'_{1}, A'_{2}, A'_{3}), and S1, S2, S3 are
	// respectively the interpolation of A'_{1}, A'_{2}, A'_{3} on <g>.
	S1, S2, S3 *iop.Polynomial

	// S full permutation, i -> S[i]
	S []int64
}

// ProvingKey stores the data needed to generate a proof
type ProvingKey struct {
	Kzg, KzgLagrange kzg.ProvingKey

	// Verifying Key is embedded into the proving key (needed by Prove)
	Vk *VerifyingKey
}

func Setup(spr *cs.SparseR1CS, srs, srsLagrange kzg.SRS) (*ProvingKey, *VerifyingKey, error) {

	var pk ProvingKey
	var vk VerifyingKey
	pk.Vk = &vk
	vk.CommitmentConstraintIndexes = internal.IntSliceToUint64Slice(spr.CommitmentInfo.CommitmentIndexes())

	// step 0: set the fft domains
	domain := initFFTDomain(spr)
	if domain.Cardinality < 2 {
		return nil, nil, fmt.Errorf("circuit has only %d constraints; unsupported by the current implementation", len(spr.Public)+spr.GetNbConstraints())
	}

	// check the size of the kzg srs.
	if len(srs.Pk.G1) < (int(domain.Cardinality) + 3) { // + 3 for the kzg.Open of blinded poly
		return nil, nil, fmt.Errorf("kzg srs is too small: got %d, need %d", len(srs.Pk.G1), domain.Cardinality+3)
	}

	// same for the lagrange form
	if len(srsLagrange.Pk.G1) != int(domain.Cardinality) {
		return nil, nil, fmt.Errorf("kzg srs lagrange is too small: got %d, need %d", len(srsLagrange.Pk.G1), domain.Cardinality)
	}

	// step 1: set the verifying key
	vk.CosetShift.Set(&domain.FrMultiplicativeGen)
	vk.Size = domain.Cardinality
	vk.SizeInv.SetUint64(vk.Size).Inverse(&vk.SizeInv)
	vk.Generator.Set(&domain.Generator)
	vk.NbPublicVariables = uint64(len(spr.Public))

	pk.Kzg.G1 = srs.Pk.G1[:int(vk.Size)+3]
	pk.KzgLagrange.G1 = srsLagrange.Pk.G1
	vk.Kzg = srs.Vk

	// step 2: ql, qr, qm, qo, qk, qcp in Lagrange Basis
	// step 3: build the permutation and build the polynomials S1, S2, S3 to encode the permutation.
	// Note: at this stage, the permutation takes in account the placeholders
	trace := NewTrace(spr, domain)

	// step 4: commit to s1, s2, s3, ql, qr, qm, qo, and (the incomplete version of) qk.
	// All the above polynomials are expressed in canonical basis afterwards. This is why
	// we save lqk before, because the prover needs to complete it in Lagrange form, and
	// then express it on the Lagrange coset basis.
	if err := vk.commitTrace(trace, domain, pk.KzgLagrange); err != nil {
		return nil, nil, err
	}

	return &pk, &vk, nil
}

// NbPublicWitness returns the expected public witness size (number of field elements)
func (vk *VerifyingKey) NbPublicWitness() int {
	return int(vk.NbPublicVariables)
}

// VerifyingKey returns pk.Vk
func (pk *ProvingKey) VerifyingKey() interface{} {
	return pk.Vk
}

// NewTrace returns a new Trace object from the constraint system.
// It fills the constant columns ql, qr, qm, qo, qk, and qcp with the
// coefficients of the constraints.
// Size is the size of the system that is next power of 2 (nb_constraints+nb_public_variables)
// The permutation is also computed and stored in the Trace.
func NewTrace(spr *cs.SparseR1CS, domain *fft.Domain) *Trace {
	var trace Trace

	size := int(domain.Cardinality)
	commitmentInfo := spr.CommitmentInfo.(constraint.PlonkCommitments)

	ql := make([]fr.Element, size)
	qr := make([]fr.Element, size)
	qm := make([]fr.Element, size)
	qo := make([]fr.Element, size)
	qk := make([]fr.Element, size)
	qcp := make([][]fr.Element, len(commitmentInfo))

	for i := 0; i < len(spr.Public); i++ { // placeholders (-PUB_INPUT_i + qk_i = 0) TODO should return error if size is inconsistent
		ql[i].SetOne().Neg(&ql[i])
		qr[i].SetZero()
		qm[i].SetZero()
		qo[i].SetZero()
		qk[i].SetZero() // → to be completed by the prover
	}
	offset := len(spr.Public)

	j := 0
	it := spr.GetSparseR1CIterator()
	for c := it.Next(); c != nil; c = it.Next() {
		ql[offset+j].Set(&spr.Coefficients[c.QL])
		qr[offset+j].Set(&spr.Coefficients[c.QR])
		qm[offset+j].Set(&spr.Coefficients[c.QM])
		qo[offset+j].Set(&spr.Coefficients[c.QO])
		qk[offset+j].Set(&spr.Coefficients[c.QC])
		j++
	}

	lagReg := iop.Form{Basis: iop.Lagrange, Layout: iop.Regular}

	trace.Ql = iop.NewPolynomial(&ql, lagReg)
	trace.Qr = iop.NewPolynomial(&qr, lagReg)
	trace.Qm = iop.NewPolynomial(&qm, lagReg)
	trace.Qo = iop.NewPolynomial(&qo, lagReg)
	trace.Qk = iop.NewPolynomial(&qk, lagReg)
	trace.Qcp = make([]*iop.Polynomial, len(qcp))

	for i := range commitmentInfo {
		qcp[i] = make([]fr.Element, size)
		for _, committed := range commitmentInfo[i].Committed {
			qcp[i][offset+committed].SetOne()
		}
		trace.Qcp[i] = iop.NewPolynomial(&qcp[i], lagReg)
	}

	// build the permutation and build the polynomials S1, S2, S3 to encode the permutation.
	// Note: at this stage, the permutation takes in account the placeholders
	nbVariables := spr.NbInternalVariables + len(spr.Public) + len(spr.Secret)
	buildPermutation(spr, &trace, nbVariables)
	s := computePermutationPolynomials(&trace, domain)
	trace.S1 = s[0]
	trace.S2 = s[1]
	trace.S3 = s[2]

	return &trace
}

// commitTrace commits to every polynomial in the trace, and put
// the commitments int the verifying key.
func (vk *VerifyingKey) commitTrace(trace *Trace, domain *fft.Domain, srsPk kzg.ProvingKey) error {

	var err error
	vk.Qcp = make([]kzg.Digest, len(trace.Qcp))
	for i := range trace.Qcp {
		if vk.Qcp[i], err = kzg.Commit(trace.Qcp[i].Coefficients(), srsPk); err != nil {
			return err
		}
	}
	if vk.Ql, err = kzg.Commit(trace.Ql.Coefficients(), srsPk); err != nil {
		return err
	}
	if vk.Qr, err = kzg.Commit(trace.Qr.Coefficients(), srsPk); err != nil {
		return err
	}
	if vk.Qm, err = kzg.Commit(trace.Qm.Coefficients(), srsPk); err != nil {
		return err
	}
	if vk.Qo, err = kzg.Commit(trace.Qo.Coefficients(), srsPk); err != nil {
		return err
	}
	if vk.Qk, err = kzg.Commit(trace.Qk.Coefficients(), srsPk); err != nil {
		return err
	}
	if vk.S[0], err = kzg.Commit(trace.S1.Coefficients(), srsPk); err != nil {
		return err
	}
	if vk.S[1], err = kzg.Commit(trace.S2.Coefficients(), srsPk); err != nil {
		return err
	}
	if vk.S[2], err = kzg.Commit(trace.S3.Coefficients(), srsPk); err != nil {
		return err
	}
	return nil
}

func initFFTDomain(spr *cs.SparseR1CS) *fft.Domain {
	nbConstraints := spr.GetNbConstraints()
	sizeSystem := uint64(nbConstraints + len(spr.Public)) // len(spr.Public) is for the placeholder constraints
	return fft.NewDomain(sizeSystem, fft.WithoutPrecompute())
}

// buildPermutation builds the Permutation associated with a circuit.
//
// The permutation s is composed of cycles of maximum length such that
//
//	s. (l∥r∥o) = (l∥r∥o)
//
// , where l∥r∥o is the concatenation of the indices of l, r, o in
// ql.l+qr.r+qm.l.r+qo.O+k = 0.
//
// The permutation is encoded as a slice s of size 3*size(l), where the
// i-th entry of l∥r∥o is sent to the s[i]-th entry, so it acts on a tab
// like this: for i in tab: tab[i] = tab[permutation[i]]
func buildPermutation(spr *cs.SparseR1CS, trace *Trace, nbVariables int) {

	// nbVariables := spr.NbInternalVariables + len(spr.Public) + len(spr.Secret)
	sizeSolution := len(trace.Ql.Coefficients())
	sizePermutation := 3 * sizeSolution

	// init permutation
	permutation := make([]int64, sizePermutation)
	for i := 0; i < len(permutation); i++ {
		permutation[i] = -1
	}

	// init LRO position -> variable_ID
	lro := make([]int, sizePermutation) // position -> variable_ID
	for i := 0; i < len(spr.Public); i++ {
		lro[i] = i // IDs of LRO associated to placeholders (only L needs to be taken care of)
	}

	offset := len(spr.Public)

	j := 0
	it := spr.GetSparseR1CIterator()
	for c := it.Next(); c != nil; c = it.Next() {
		lro[offset+j] = int(c.XA)
		lro[sizeSolution+offset+j] = int(c.XB)
		lro[2*sizeSolution+offset+j] = int(c.XC)

		j++
	}

	// init cycle:
	// map ID -> last position the ID was seen
	cycle := make([]int64, nbVariables)
	for i := 0; i < len(cycle); i++ {
		cycle[i] = -1
	}

	for i := 0; i < len(lro); i++ {
		if cycle[lro[i]] != -1 {
			// if != -1, it means we already encountered this value
			// so we need to set the corresponding permutation index.
			permutation[i] = cycle[lro[i]]
		}
		cycle[lro[i]] = int64(i)
	}

	// complete the Permutation by filling the first IDs encountered
	for i := 0; i < sizePermutation; i++ {
		if permutation[i] == -1 {
			permutation[i] = cycle[lro[i]]
		}
	}

	trace.S = permutation
}

// computePermutationPolynomials computes the LDE (Lagrange basis) of the permutation.
// We let the permutation act on <g> || u<g> || u^{2}<g>, split the result in 3 parts,
// and interpolate each of the 3 parts on <g>.
func computePermutationPolynomials(trace *Trace, domain *fft.Domain) [3]*iop.Polynomial {

	nbElmts := int(domain.Cardinality)

	var res [3]*iop.Polynomial

	// Lagrange form of ID
	evaluationIDSmallDomain := getSupportPermutation(domain)

	// Lagrange form of S1, S2, S3
	s1Canonical := make([]fr.Element, nbElmts)
	s2Canonical := make([]fr.Element, nbElmts)
	s3Canonical := make([]fr.Element, nbElmts)
	for i := 0; i < nbElmts; i++ {
		s1Canonical[i].Set(&evaluationIDSmallDomain[trace.S[i]])
		s2Canonical[i].Set(&evaluationIDSmallDomain[trace.S[nbElmts+i]])
		s3Canonical[i].Set(&evaluationIDSmallDomain[trace.S[2*nbElmts+i]])
	}

	lagReg := iop.Form{Basis: iop.Lagrange, Layout: iop.Regular}
	res[0] = iop.NewPolynomial(&s1Canonical, lagReg)
	res[1] = iop.NewPolynomial(&s2Canonical, lagReg)
	res[2] = iop.NewPolynomial(&s3Canonical, lagReg)

	return res
}

// getSupportPermutation returns the support on which the permutation acts, it is
// <g> || u<g> || u^{2}<g>
func getSupportPermutation(domain *fft.Domain) []fr.Element {

	res := make([]fr.Element, 3*domain.Cardinality)

	res[0].SetOne()
	res[domain.Cardinality].Set(&domain.FrMultiplicativeGen)
	res[2*domain.Cardinality].Square(&domain.FrMultiplicativeGen)

	for i := uint64(1); i < domain.Cardinality; i++ {
		res[i].Mul(&res[i-1], &domain.Generator)
		res[domain.Cardinality+i].Mul(&res[domain.Cardinality+i-1], &domain.Generator)
		res[2*domain.Cardinality+i].Mul(&res[2*domain.Cardinality+i-1], &domain.Generator)
	}

	return res
}
