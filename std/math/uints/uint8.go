// Package uints implements optimised byte and long integer operations.
//
// Usually arithmetic in a circuit is performed in the native field, which is of
// prime order. However, for compatibility with native operations we rely on
// operating on smaller primitive types as 8-bit, 32-bit and 64-bit integer.
// Naively, these operations have to be implemented bitwise as there are no
// closed equations for boolean operations (XOR, AND, OR).
//
// However, the bitwise approach is very inefficient and leads to several
// constraints per bit. Accumulating over a long integer, it leads to very
// inefficients circuits.
//
// This package performs boolean operations using lookup tables on bytes. So,
// long integers are split into 4 or 8 bytes and we perform the operations
// bytewise. In the lookup tables, we store results for all possible 2^8×2^8
// inputs. With this approach, every bytewise operation costs as single lookup,
// which depending on the backend is relatively cheap (one to three
// constraints).
//
// NB! The package is still work in progress. The interfaces and implementation
// details most certainly changes over time. We cannot ensure the soundness of
// the operations.
package uints

import (
	"fmt"
	"math/big"
	"math/bits"

	"github.com/BaoNinh2808/gnark/std/internal/logderivprecomp"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/math/bitslice"
	"github.com/consensys/gnark/std/math/cmp"
	"github.com/consensys/gnark/std/rangecheck"
)

// TODO: if internal then enforce range check!

// TODO: all operations can take rand linear combinations instead. Then instead
// of one check can perform multiple at the same time.

// TODO: implement versions which take multiple inputs. Maybe can combine multiple together

// TODO: instantiate tables only when we first query. Maybe do not need to build!

// TODO: maybe can store everything in a single table? Later! Or if we have a
// lot of queries then makes sense to extract into separate table?

// TODO: in ValueOf ensure consistency

// TODO: distinguish between when we set constant in-circuit or witness
// assignment. For constant we don't have to range check but for witness
// assignment we have to.

// TODO: add something which allows to store array in native element

// TODO: add methods for checking if U8/Long is constant.

// TODO: should something for byte-only ops. Implement a type and then embed it in BinaryField

// TODO: add helper method to call hints which allows to pass in uint8s (bytes)
// and returns bytes. Then can to byte array manipulation nicely. It is useful
// for X509. For the implementation we want to pack as much bytes into a field
// element as possible.

// TODO: methods for converting uint array into emulated element and native
// element. Most probably should add the implementation for non-native in its
// package, but for native we should add it here.

type U8 struct {
	Val      frontend.Variable
	internal bool
}

// GnarkInitHook describes how to initialise the element.
func (e *U8) GnarkInitHook() {
	if e.Val == nil {
		e.Val = 0
		e.internal = false // we need to constrain in later.
	}
}

type U64 [8]U8
type U32 [4]U8

type Long interface{ U32 | U64 }

type BinaryField[T U32 | U64] struct {
	api        frontend.API
	u8cmpApi   *cmp.BoundedComparator
	xorT, andT *logderivprecomp.Precomputed
	rchecker   frontend.Rangechecker
	allOne     U8
}

func New[T Long](api frontend.API) (*BinaryField[T], error) {
	xorT, err := logderivprecomp.New(api, xorHint, []uint{8})
	if err != nil {
		return nil, fmt.Errorf("new xor table: %w", err)
	}
	andT, err := logderivprecomp.New(api, andHint, []uint{8})
	if err != nil {
		return nil, fmt.Errorf("new and table: %w", err)
	}
	rchecker := rangecheck.New(api)

	u8cmpApi := cmp.NewBoundedComparator(api, big.NewInt(255), false) //in u8 we can have 2^8 values --> upperAbsDiff = 11111111b - 00000000b = 255
	bf := &BinaryField[T]{
		api:      api,
		u8cmpApi: u8cmpApi,
		xorT:     xorT,
		andT:     andT,
		rchecker: rchecker,
	}
	// TODO: this is const. add way to init constants
	allOne := bf.ByteValueOf(0xff)
	bf.allOne = allOne
	return bf, nil
}

func NewU8(v uint8) U8 {
	// TODO: don't have to check constants
	return U8{Val: v, internal: true}
}

func NewU32(v uint32) U32 {
	return [4]U8{
		NewU8(uint8((v >> (0 * 8)) & 0xff)),
		NewU8(uint8((v >> (1 * 8)) & 0xff)),
		NewU8(uint8((v >> (2 * 8)) & 0xff)),
		NewU8(uint8((v >> (3 * 8)) & 0xff)),
	}
}

func NewU64(v uint64) U64 {
	return [8]U8{
		NewU8(uint8((v >> (0 * 8)) & 0xff)),
		NewU8(uint8((v >> (1 * 8)) & 0xff)),
		NewU8(uint8((v >> (2 * 8)) & 0xff)),
		NewU8(uint8((v >> (3 * 8)) & 0xff)),
		NewU8(uint8((v >> (4 * 8)) & 0xff)),
		NewU8(uint8((v >> (5 * 8)) & 0xff)),
		NewU8(uint8((v >> (6 * 8)) & 0xff)),
		NewU8(uint8((v >> (7 * 8)) & 0xff)),
	}
}

func NewU8Array(v []uint8) []U8 {
	ret := make([]U8, len(v))
	for i := range v {
		ret[i] = NewU8(v[i])
	}
	return ret
}

func NewU32Array(v []uint32) []U32 {
	ret := make([]U32, len(v))
	for i := range v {
		ret[i] = NewU32(v[i])
	}
	return ret
}

func NewU64Array(v []uint64) []U64 {
	ret := make([]U64, len(v))
	for i := range v {
		ret[i] = NewU64(v[i])
	}
	return ret
}

func (bf *BinaryField[T]) ByteValueOf(a frontend.Variable) U8 {
	bf.rchecker.Check(a, 8)
	return U8{Val: a, internal: true}
}

func (bf *BinaryField[T]) ValueOf(a frontend.Variable) T {
	var r T
	bts, err := bf.api.Compiler().NewHint(toBytes, len(r), len(r), a)
	if err != nil {
		panic(err)
	}

	for i := range bts {
		r[i] = bf.ByteValueOf(bts[i])
	}
	expectedValue := bf.ToValue(r)
	bf.api.AssertIsEqual(a, expectedValue)

	return r
}

func (bf *BinaryField[T]) ToValue(a T) frontend.Variable {
	v := make([]frontend.Variable, bf.lenBts())
	for i := range v {
		v[i] = bf.api.Mul(a[i].Val, 1<<(i*8))
	}
	vv := bf.api.Add(v[0], v[1], v[2:]...)
	return vv
}

func (bf *BinaryField[T]) PackMSB(a ...U8) T {
	var ret T
	for i := range a {
		ret[len(a)-i-1] = a[i]
	}
	return ret
}

func (bf *BinaryField[T]) PackLSB(a ...U8) T {
	var ret T
	for i := range a {
		ret[i] = a[i]
	}
	return ret
}

func (bf *BinaryField[T]) UnpackMSB(a T) []U8 {
	ret := make([]U8, bf.lenBts())
	for i := 0; i < len(ret); i++ {
		ret[len(a)-i-1] = a[i]
	}
	return ret
}

func (bf *BinaryField[T]) UnpackLSB(a T) []U8 {
	// cannot deduce that a can be cast to []U8
	ret := make([]U8, bf.lenBts())
	for i := 0; i < len(ret); i++ {
		ret[i] = a[i]
	}
	return ret
}

func (bf *BinaryField[T]) twoArgFn(tbl *logderivprecomp.Precomputed, a ...U8) U8 {
	ret := tbl.Query(a[0].Val, a[1].Val)[0]
	for i := 2; i < len(a); i++ {
		ret = tbl.Query(ret, a[i].Val)[0]
	}
	return U8{Val: ret}
}

func (bf *BinaryField[T]) twoArgWideFn(tbl *logderivprecomp.Precomputed, a ...T) T {
	var r T
	for i, v := range reslice(a) {
		r[i] = bf.twoArgFn(tbl, v...)
	}
	return r
}

func (bf *BinaryField[T]) And(a ...T) T { return bf.twoArgWideFn(bf.andT, a...) }
func (bf *BinaryField[T]) Xor(a ...T) T { return bf.twoArgWideFn(bf.xorT, a...) }

func (bf *BinaryField[T]) not(a U8) U8 {
	ret := bf.xorT.Query(a.Val, bf.allOne.Val)
	return U8{Val: ret[0]}
}

func (bf *BinaryField[T]) Not(a T) T {
	var r T
	for i := 0; i < len(a); i++ {
		r[i] = bf.not(a[i])
	}
	return r
}

func (bf *BinaryField[T]) Add(a ...T) T {
	tLen := bf.lenBts() * 8
	inLen := len(a)
	va := make([]frontend.Variable, inLen)
	for i := range a {
		va[i] = bf.ToValue(a[i])
	}
	vres := bf.api.Add(va[0], va[1], va[2:]...)
	maxBitlen := bits.Len(uint(inLen)) + tLen
	// bitslice.Partition below checks that the input is less than 2^maxBitlen and that we have omitted carry correctly
	vreslow, _ := bitslice.Partition(bf.api, vres, uint(tLen), bitslice.WithNbDigits(maxBitlen), bitslice.WithUnconstrainedOutputs())
	res := bf.ValueOf(vreslow)
	return res
}

func (bf *BinaryField[T]) Lrot(a T, c int) T {
	l := bf.lenBts()
	if c < 0 {
		c = l*8 + c
	}
	shiftBl := c / 8
	shiftBt := c % 8
	revShiftBt := 8 - shiftBt
	if revShiftBt == 8 {
		revShiftBt = 0
	}
	partitioned := make([][2]frontend.Variable, l)
	for i := range partitioned {
		lower, upper := bitslice.Partition(bf.api, a[i].Val, uint(revShiftBt), bitslice.WithNbDigits(8))
		partitioned[i] = [2]frontend.Variable{lower, upper}
	}
	var ret T
	for i := 0; i < l; i++ {
		if shiftBt != 0 {
			ret[(i+shiftBl)%l].Val = bf.api.Add(bf.api.Mul(1<<(shiftBt), partitioned[i][0]), partitioned[(i+l-1)%l][1])
		} else {
			ret[(i+shiftBl)%l].Val = partitioned[i][1]
		}
	}
	return ret
}

func (bf *BinaryField[T]) Rshift(a T, c int) T {
	lenB := bf.lenBts()
	shiftBl := c / 8
	shiftBt := c % 8
	partitioned := make([][2]frontend.Variable, lenB-shiftBl)
	for i := range partitioned {
		lower, upper := bitslice.Partition(bf.api, a[i+shiftBl].Val, uint(shiftBt), bitslice.WithNbDigits(8))
		partitioned[i] = [2]frontend.Variable{lower, upper}
	}
	var ret T
	for i := 0; i < bf.lenBts()-shiftBl-1; i++ {
		if shiftBt != 0 {
			ret[i].Val = bf.api.Add(partitioned[i][1], bf.api.Mul(1<<(8-shiftBt), partitioned[i+1][0]))
		} else {
			ret[i].Val = partitioned[i][1]
		}
	}
	ret[lenB-shiftBl-1].Val = partitioned[lenB-shiftBl-1][1]
	for i := lenB - shiftBl; i < lenB; i++ {
		ret[i] = NewU8(0)
	}
	return ret
}

func (bf *BinaryField[T]) ByteAssertEq(a, b U8) {
	bf.api.AssertIsEqual(a.Val, b.Val)
}

// AddU8 adds multiple U8 values and returns a U8.
func (bf *BinaryField[T]) AddU8(values ...U8) U8 {
	if len(values) == 0 {
		return U8{Val: 0}
	}
	sum := values[0].Val
	for i := 1; i < len(values); i++ {
		sum = bf.api.Add(sum, values[i].Val)
	}
	return U8{Val: sum}
}

func (bf *BinaryField[T]) AssertEq(a, b T) {
	for i := 0; i < bf.lenBts(); i++ {
		bf.ByteAssertEq(a[i], b[i])
	}
}

func (bf *BinaryField[T]) ByteAssertIsLess(a, b U8) {
	bf.u8cmpApi.AssertIsLess(a.Val, b.Val)
}

func (bf *BinaryField[T]) ByteAssertIsLessEq(a, b U8) {
	bf.u8cmpApi.AssertIsLessEq(a.Val, b.Val)
}

func (bf *BinaryField[T]) IsEqualU8(a, b U8) frontend.Variable {
	return bf.api.IsZero(bf.api.Sub(a.Val, b.Val))
}

func (bf *BinaryField[T]) IsLessEqualU8(a, b U8) frontend.Variable {
	return bf.u8cmpApi.IsLessEq(a.Val, b.Val)
}

func (bf *BinaryField[T]) CompareU8(a, b U8) frontend.Variable {
	// 1 if a > b
	// 0 if a = b
	// -1 if a < b

	isEqual := bf.IsEqualU8(a, b)
	isLess := bf.u8cmpApi.IsLess(a.Val, b.Val)

	// Return 1 if a > b, 0 if a = b, -1 if a < b
	return bf.api.Select(isEqual, 0, bf.api.Select(isLess, -1, 1))
}

func (bf *BinaryField[T]) IsEqual(a, b T) frontend.Variable {
	lenB := bf.lenBts()
	// create a array of frontend.Variable with lenght lenB
	isEqual := make([]frontend.Variable, lenB)

	for i := 0; i < lenB; i++ {
		isEqual[i] = bf.IsEqualU8(a[i], b[i])
	}

	res := frontend.Variable(1)
	for i := 0; i < lenB; i++ {
		res = bf.api.And(res, isEqual[i])
	}

	return res
}

func (bf *BinaryField[T]) IsLessU8(a, b U8) frontend.Variable {
	return bf.u8cmpApi.IsLess(a.Val, b.Val)
}

func (bf *BinaryField[T]) IsLess(a, b T) frontend.Variable {
	fmt.Println("a : ", a)
	lenB := bf.lenBts()

	// create a array of frontend.Variable with lenght lenB
	isLess := make([]frontend.Variable, lenB)
	isEqual := make([]frontend.Variable, lenB)

	isLess[0] = bf.IsLessU8(a[lenB-1], b[lenB-1])
	isEqual[0] = bf.IsEqualU8(a[lenB-1], b[lenB-1])

	for i := 1; i < lenB; i++ {
		isLess[i] = bf.api.Select(isLess[i-1], isLess[i-1], bf.IsLessU8(a[lenB-1-i], b[lenB-1-i]))
		isEqual[i] = bf.api.Select(bf.api.IsZero(isEqual[i-1]), isEqual[i-1], bf.IsEqualU8(a[lenB-1-i], b[lenB-1-i]))
	}

	fmt.Println("isLess: ", isLess)

	//assert isLess != 0 (because there is a case that a = b ==> isLess = {0, 0, 0, 0, 0, 0, 0, 0} & isEqual = {1, 1, 1, 1, 1, 1, 1, 1} ==> xorValue = {1, 1, 1, 1, 1, 1, 1, 1})
	sum := frontend.Variable(0)
	for i := 0; i < len(isLess); i++ {
		sum = bf.api.Add(sum, isLess[i])
	}
	isLessAllZero := bf.api.IsZero(sum)
	fmt.Println("isLessAllZero: ", isLessAllZero)

	xorValue := make([]frontend.Variable, lenB)
	for i := 0; i < lenB; i++ {
		xorValue[i] = bf.api.Xor(isLess[i], isEqual[i])
	}

	andXorValue := frontend.Variable(0)
	for i := 0; i < lenB; i++ {
		andXorValue = bf.api.Add(andXorValue, xorValue[i])
	}
	isXorValueAllOne := bf.api.IsZero(bf.api.Sub(andXorValue, lenB))

	res := bf.api.Select(isLessAllZero, 0, isXorValueAllOne)
	fmt.Println("res: ", res)
	return res
}

func (bf *BinaryField[T]) AssertIsLess(a, b T) {
	lenB := bf.lenBts()
	// create a array of frontend.Variable with lenght lenB
	isLess := make([]frontend.Variable, lenB)
	isEqual := make([]frontend.Variable, lenB)

	isLess[0] = bf.IsLessU8(a[lenB-1], b[lenB-1])
	isEqual[0] = bf.IsEqualU8(a[lenB-1], b[lenB-1])

	for i := 1; i < lenB; i++ {
		isLess[i] = bf.api.Select(isLess[i-1], isLess[i-1], bf.IsLessU8(a[lenB-1-i], b[lenB-1-i]))
		isEqual[i] = bf.api.Select(bf.api.IsZero(isEqual[i-1]), isEqual[i-1], bf.IsEqualU8(a[lenB-1-i], b[lenB-1-i]))
	}

	//assert isLess != 0 (because there is a case that a = b ==> isLess = {0, 0, 0, 0, 0, 0, 0, 0} & isEqual = {1, 1, 1, 1, 1, 1, 1, 1} ==> xorValue = {1, 1, 1, 1, 1, 1, 1, 1})
	sum := frontend.Variable(0)
	for i := 0; i < len(isLess); i++ {
		sum = bf.api.Add(sum, isLess[i])
	}
	bf.api.AssertIsDifferent(sum, 0)

	//assert xorValue = xor(isLess, isEqual) = {1, 1, 1, 1, 1, 1, 1, 1}
	xorValue := make([]frontend.Variable, lenB)
	for i := 0; i < lenB; i++ {
		xorValue[i] = bf.api.Xor(isLess[i], isEqual[i])
	}

	for i := 0; i < lenB; i++ {
		bf.api.AssertIsEqual(xorValue[i], 1)
	}
}

func (bf *BinaryField[T]) lenBts() int {
	var a T
	return len(a)
}

func reslice[T U32 | U64](in []T) [][]U8 {
	if len(in) == 0 {
		panic("zero-length input")
	}
	ret := make([][]U8, len(in[0]))
	for i := range ret {
		ret[i] = make([]U8, len(in))
	}
	for i := 0; i < len(in); i++ {
		for j := 0; j < len(in[0]); j++ {
			ret[j][i] = in[i][j]
		}
	}
	return ret
}
