package fields_bls12381

import (
	"math/big"

	bls12381 "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fp"
	"github.com/consensys/gnark/constraint/solver"
	"github.com/consensys/gnark/std/math/emulated"
)

func init() {
	solver.RegisterHint(GetHints()...)
}

// GetHints returns all hint functions used in the package.
func GetHints() []solver.Hint {
	return []solver.Hint{
		// E2
		divE2Hint,
		inverseE2Hint,
		// E6
		divE6Hint,
		inverseE6Hint,
		squareTorusHint,
		divE6By6Hint,
		// E12
		divE12Hint,
		inverseE12Hint,
		finalExpHint,
	}
}

func inverseE2Hint(nativeMod *big.Int, nativeInputs, nativeOutputs []*big.Int) error {
	return emulated.UnwrapHint(nativeInputs, nativeOutputs,
		func(mod *big.Int, inputs, outputs []*big.Int) error {
			var a, c bls12381.E2

			a.A0.SetBigInt(inputs[0])
			a.A1.SetBigInt(inputs[1])

			c.Inverse(&a)

			c.A0.BigInt(outputs[0])
			c.A1.BigInt(outputs[1])

			return nil
		})
}

func divE2Hint(nativeMod *big.Int, nativeInputs, nativeOutputs []*big.Int) error {
	return emulated.UnwrapHint(nativeInputs, nativeOutputs,
		func(mod *big.Int, inputs, outputs []*big.Int) error {
			var a, b, c bls12381.E2

			a.A0.SetBigInt(inputs[0])
			a.A1.SetBigInt(inputs[1])
			b.A0.SetBigInt(inputs[2])
			b.A1.SetBigInt(inputs[3])

			c.Inverse(&b).Mul(&c, &a)

			c.A0.BigInt(outputs[0])
			c.A1.BigInt(outputs[1])

			return nil
		})
}

// E6 hints
func inverseE6Hint(nativeMod *big.Int, nativeInputs, nativeOutputs []*big.Int) error {
	return emulated.UnwrapHint(nativeInputs, nativeOutputs,
		func(mod *big.Int, inputs, outputs []*big.Int) error {
			var a, c bls12381.E6

			a.B0.A0.SetBigInt(inputs[0])
			a.B0.A1.SetBigInt(inputs[1])
			a.B1.A0.SetBigInt(inputs[2])
			a.B1.A1.SetBigInt(inputs[3])
			a.B2.A0.SetBigInt(inputs[4])
			a.B2.A1.SetBigInt(inputs[5])

			c.Inverse(&a)

			c.B0.A0.BigInt(outputs[0])
			c.B0.A1.BigInt(outputs[1])
			c.B1.A0.BigInt(outputs[2])
			c.B1.A1.BigInt(outputs[3])
			c.B2.A0.BigInt(outputs[4])
			c.B2.A1.BigInt(outputs[5])

			return nil
		})
}

func divE6Hint(nativeMod *big.Int, nativeInputs, nativeOutputs []*big.Int) error {
	return emulated.UnwrapHint(nativeInputs, nativeOutputs,
		func(mod *big.Int, inputs, outputs []*big.Int) error {
			var a, b, c bls12381.E6

			a.B0.A0.SetBigInt(inputs[0])
			a.B0.A1.SetBigInt(inputs[1])
			a.B1.A0.SetBigInt(inputs[2])
			a.B1.A1.SetBigInt(inputs[3])
			a.B2.A0.SetBigInt(inputs[4])
			a.B2.A1.SetBigInt(inputs[5])

			b.B0.A0.SetBigInt(inputs[6])
			b.B0.A1.SetBigInt(inputs[7])
			b.B1.A0.SetBigInt(inputs[8])
			b.B1.A1.SetBigInt(inputs[9])
			b.B2.A0.SetBigInt(inputs[10])
			b.B2.A1.SetBigInt(inputs[11])

			c.Inverse(&b).Mul(&c, &a)

			c.B0.A0.BigInt(outputs[0])
			c.B0.A1.BigInt(outputs[1])
			c.B1.A0.BigInt(outputs[2])
			c.B1.A1.BigInt(outputs[3])
			c.B2.A0.BigInt(outputs[4])
			c.B2.A1.BigInt(outputs[5])

			return nil
		})
}

func squareTorusHint(nativeMod *big.Int, nativeInputs, nativeOutputs []*big.Int) error {
	return emulated.UnwrapHint(nativeInputs, nativeOutputs,
		func(mod *big.Int, inputs, outputs []*big.Int) error {
			var a, c bls12381.E6

			a.B0.A0.SetBigInt(inputs[0])
			a.B0.A1.SetBigInt(inputs[1])
			a.B1.A0.SetBigInt(inputs[2])
			a.B1.A1.SetBigInt(inputs[3])
			a.B2.A0.SetBigInt(inputs[4])
			a.B2.A1.SetBigInt(inputs[5])

			_c := a.DecompressTorus()
			_c.CyclotomicSquare(&_c)
			c, _ = _c.CompressTorus()

			c.B0.A0.BigInt(outputs[0])
			c.B0.A1.BigInt(outputs[1])
			c.B1.A0.BigInt(outputs[2])
			c.B1.A1.BigInt(outputs[3])
			c.B2.A0.BigInt(outputs[4])
			c.B2.A1.BigInt(outputs[5])

			return nil
		})
}

func divE6By6Hint(nativeMod *big.Int, nativeInputs, nativeOutputs []*big.Int) error {
	return emulated.UnwrapHint(nativeInputs, nativeOutputs,
		func(mod *big.Int, inputs, outputs []*big.Int) error {
			var a, c bls12381.E6

			a.B0.A0.SetBigInt(inputs[0])
			a.B0.A1.SetBigInt(inputs[1])
			a.B1.A0.SetBigInt(inputs[2])
			a.B1.A1.SetBigInt(inputs[3])
			a.B2.A0.SetBigInt(inputs[4])
			a.B2.A1.SetBigInt(inputs[5])

			var sixInv fp.Element
			sixInv.SetString("6")
			sixInv.Inverse(&sixInv)
			c.B0.MulByElement(&a.B0, &sixInv)
			c.B1.MulByElement(&a.B1, &sixInv)
			c.B2.MulByElement(&a.B2, &sixInv)

			c.B0.A0.BigInt(outputs[0])
			c.B0.A1.BigInt(outputs[1])
			c.B1.A0.BigInt(outputs[2])
			c.B1.A1.BigInt(outputs[3])
			c.B2.A0.BigInt(outputs[4])
			c.B2.A1.BigInt(outputs[5])

			return nil
		})
}

// E12 hints
func inverseE12Hint(nativeMod *big.Int, nativeInputs, nativeOutputs []*big.Int) error {
	return emulated.UnwrapHint(nativeInputs, nativeOutputs,
		func(mod *big.Int, inputs, outputs []*big.Int) error {
			var a, c bls12381.E12

			a.C0.B0.A0.SetBigInt(inputs[0])
			a.C0.B0.A1.SetBigInt(inputs[1])
			a.C0.B1.A0.SetBigInt(inputs[2])
			a.C0.B1.A1.SetBigInt(inputs[3])
			a.C0.B2.A0.SetBigInt(inputs[4])
			a.C0.B2.A1.SetBigInt(inputs[5])
			a.C1.B0.A0.SetBigInt(inputs[6])
			a.C1.B0.A1.SetBigInt(inputs[7])
			a.C1.B1.A0.SetBigInt(inputs[8])
			a.C1.B1.A1.SetBigInt(inputs[9])
			a.C1.B2.A0.SetBigInt(inputs[10])
			a.C1.B2.A1.SetBigInt(inputs[11])

			c.Inverse(&a)

			c.C0.B0.A0.BigInt(outputs[0])
			c.C0.B0.A1.BigInt(outputs[1])
			c.C0.B1.A0.BigInt(outputs[2])
			c.C0.B1.A1.BigInt(outputs[3])
			c.C0.B2.A0.BigInt(outputs[4])
			c.C0.B2.A1.BigInt(outputs[5])
			c.C1.B0.A0.BigInt(outputs[6])
			c.C1.B0.A1.BigInt(outputs[7])
			c.C1.B1.A0.BigInt(outputs[8])
			c.C1.B1.A1.BigInt(outputs[9])
			c.C1.B2.A0.BigInt(outputs[10])
			c.C1.B2.A1.BigInt(outputs[11])

			return nil
		})
}

func divE12Hint(nativeMod *big.Int, nativeInputs, nativeOutputs []*big.Int) error {
	return emulated.UnwrapHint(nativeInputs, nativeOutputs,
		func(mod *big.Int, inputs, outputs []*big.Int) error {
			var a, b, c bls12381.E12

			a.C0.B0.A0.SetBigInt(inputs[0])
			a.C0.B0.A1.SetBigInt(inputs[1])
			a.C0.B1.A0.SetBigInt(inputs[2])
			a.C0.B1.A1.SetBigInt(inputs[3])
			a.C0.B2.A0.SetBigInt(inputs[4])
			a.C0.B2.A1.SetBigInt(inputs[5])
			a.C1.B0.A0.SetBigInt(inputs[6])
			a.C1.B0.A1.SetBigInt(inputs[7])
			a.C1.B1.A0.SetBigInt(inputs[8])
			a.C1.B1.A1.SetBigInt(inputs[9])
			a.C1.B2.A0.SetBigInt(inputs[10])
			a.C1.B2.A1.SetBigInt(inputs[11])

			b.C0.B0.A0.SetBigInt(inputs[12])
			b.C0.B0.A1.SetBigInt(inputs[13])
			b.C0.B1.A0.SetBigInt(inputs[14])
			b.C0.B1.A1.SetBigInt(inputs[15])
			b.C0.B2.A0.SetBigInt(inputs[16])
			b.C0.B2.A1.SetBigInt(inputs[17])
			b.C1.B0.A0.SetBigInt(inputs[18])
			b.C1.B0.A1.SetBigInt(inputs[19])
			b.C1.B1.A0.SetBigInt(inputs[20])
			b.C1.B1.A1.SetBigInt(inputs[21])
			b.C1.B2.A0.SetBigInt(inputs[22])
			b.C1.B2.A1.SetBigInt(inputs[23])

			c.Inverse(&b).Mul(&c, &a)

			c.C0.B0.A0.BigInt(outputs[0])
			c.C0.B0.A1.BigInt(outputs[1])
			c.C0.B1.A0.BigInt(outputs[2])
			c.C0.B1.A1.BigInt(outputs[3])
			c.C0.B2.A0.BigInt(outputs[4])
			c.C0.B2.A1.BigInt(outputs[5])
			c.C1.B0.A0.BigInt(outputs[6])
			c.C1.B0.A1.BigInt(outputs[7])
			c.C1.B1.A0.BigInt(outputs[8])
			c.C1.B1.A1.BigInt(outputs[9])
			c.C1.B2.A0.BigInt(outputs[10])
			c.C1.B2.A1.BigInt(outputs[11])

			return nil
		})
}

func finalExpHint(nativeMod *big.Int, nativeInputs, nativeOutputs []*big.Int) error {
	// This is inspired from https://eprint.iacr.org/2024/640.pdf
	// and based on a personal communication with the author Andrija Novakovic.
	return emulated.UnwrapHint(nativeInputs, nativeOutputs,
		func(mod *big.Int, inputs, outputs []*big.Int) error {
			var millerLoop bls12381.E12

			millerLoop.C0.B0.A0.SetBigInt(inputs[0])
			millerLoop.C0.B0.A1.SetBigInt(inputs[1])
			millerLoop.C0.B1.A0.SetBigInt(inputs[2])
			millerLoop.C0.B1.A1.SetBigInt(inputs[3])
			millerLoop.C0.B2.A0.SetBigInt(inputs[4])
			millerLoop.C0.B2.A1.SetBigInt(inputs[5])
			millerLoop.C1.B0.A0.SetBigInt(inputs[6])
			millerLoop.C1.B0.A1.SetBigInt(inputs[7])
			millerLoop.C1.B1.A0.SetBigInt(inputs[8])
			millerLoop.C1.B1.A1.SetBigInt(inputs[9])
			millerLoop.C1.B2.A0.SetBigInt(inputs[10])
			millerLoop.C1.B2.A1.SetBigInt(inputs[11])

			var y, wj, rootPthInverse, root27thInverse, residueWitness, scalingFactor bls12381.E12
			var ord, pw, exp, h3, v, vInv, s, p big.Int
			// p = (1-x)/3
			p.SetString("5044125407647214251", 10)
			// h3 = ((q**12 - 1) / r) / (27 * p)
			h3.SetString("2366356426548243601069753987687709088104621721678962410379583120840019275952471579477684846670499039076873213559162845121989217658133790336552276567078487633052653005423051750848782286407340332979263075575489766963251914185767058009683318020965829271737924625612375201545022326908440428522712877494557944965298566001441468676802477524234094954960009227631543471415676620753242466901942121887152806837594306028649150255258504417829961387165043999299071444887652375514277477719817175923289019181393803729926249507024121957184340179467502106891835144220611408665090353102353194448552304429530104218473070114105759487413726485729058069746063140422361472585604626055492939586602274983146215294625774144156395553405525711143696689756441298365274341189385646499074862712688473936093315628166094221735056483459332831845007196600723053356837526749543765815988577005929923802636375670820616189737737304893769679803809426304143627363860243558537831172903494450556755190448279875942974830469855835666815454271389438587399739607656399812689280234103023464545891697941661992848552456326290792224091557256350095392859243101357349751064730561345062266850238821755009430903520645523345000326783803935359711318798844368754833295302563158150573540616830138810935344206231367357992991289265295323280", 10)

			// 1. get pth-root inverse
			v.Mul(&h3, big.NewInt(27))
			wj.Exp(millerLoop, &v)
			if wj.IsOne() {
				rootPthInverse.SetOne()
			} else {
				vInv.ModInverse(&v, &p)
				s.Neg(&vInv).Mod(&s, &p)
				rootPthInverse.Exp(wj, &s)
			}

			// 2.1. get order of 3rd primitive root
			exp.Mul(&p, &h3)
			y.Exp(millerLoop, &exp)
			if y.IsOne() {
				pw.SetUint64(0)
			}
			y.Exp(y, big.NewInt(3))
			if y.IsOne() {
				pw.SetUint64(1)
			}
			y.Exp(y, big.NewInt(3))
			if y.IsOne() {
				pw.SetUint64(2)
			}
			y.Exp(y, big.NewInt(3))
			if y.IsOne() {
				pw.SetUint64(3)
			}

			// 2.2. get 27th root inverse
			if pw.Uint64() == 0 {
				root27thInverse.SetOne()
			} else {
				ord.Exp(big.NewInt(3), &pw, nil)
				v.Mul(&p, &h3)
				wj.Exp(millerLoop, &v)
				vInv.ModInverse(&v, &ord)
				s.Neg(&vInv).Mod(&s, &ord)
				root27thInverse.Exp(wj, &s)
			}

			// return the scaling factor so that millerLoop * scalingFactor is of order h3
			scalingFactor.Mul(&rootPthInverse, &root27thInverse)

			scalingFactor.C0.B0.A0.BigInt(outputs[0])
			scalingFactor.C0.B0.A1.BigInt(outputs[1])
			scalingFactor.C0.B1.A0.BigInt(outputs[2])
			scalingFactor.C0.B1.A1.BigInt(outputs[3])
			scalingFactor.C0.B2.A0.BigInt(outputs[4])
			scalingFactor.C0.B2.A1.BigInt(outputs[5])
			scalingFactor.C1.B0.A0.BigInt(outputs[6])
			scalingFactor.C1.B0.A1.BigInt(outputs[7])
			scalingFactor.C1.B1.A0.BigInt(outputs[8])
			scalingFactor.C1.B1.A1.BigInt(outputs[9])
			scalingFactor.C1.B2.A0.BigInt(outputs[10])
			scalingFactor.C1.B2.A1.BigInt(outputs[11])

			millerLoop.Mul(&millerLoop, &scalingFactor)

			// 3. get the witness residue
			// lambda = q - u
			var lambda big.Int
			lambda.SetString("4002409555221667393417789825735904156556882819939007885332058136124031650490837864442687629129030796414117214202539", 10)
			exp.ModInverse(&lambda, &h3)
			residueWitness.Exp(millerLoop, &exp)

			// return the witness residue
			residueWitness.C0.B0.A0.BigInt(outputs[12])
			residueWitness.C0.B0.A1.BigInt(outputs[13])
			residueWitness.C0.B1.A0.BigInt(outputs[14])
			residueWitness.C0.B1.A1.BigInt(outputs[15])
			residueWitness.C0.B2.A0.BigInt(outputs[16])
			residueWitness.C0.B2.A1.BigInt(outputs[17])
			residueWitness.C1.B0.A0.BigInt(outputs[18])
			residueWitness.C1.B0.A1.BigInt(outputs[19])
			residueWitness.C1.B1.A0.BigInt(outputs[20])
			residueWitness.C1.B1.A1.BigInt(outputs[21])
			residueWitness.C1.B2.A0.BigInt(outputs[22])
			residueWitness.C1.B2.A1.BigInt(outputs[23])

			return nil
		})
}
