package diff_test

import (
	"fmt"
	"math"
	"testing"
	"unsafe"

	"kr.dev/diff"
)

var NaN = math.NaN()

func TestEqual(t *testing.T) {
	// This function is just a convenient way to populate
	// the same value in both spots. But for some tests,
	// it's important to have two distinct pointers
	// (either explicit pointers or internal pointers in
	// maps or slices) pointing to equal data, so in those
	// cases we avoid this function.
	ab := func(v any) [2]any { return [2]any{v, v} }

	var cases = [][2]any{
		{[1]int{0}, [1]int{0}},
		ab(struct{ V int }{0}),
		ab(struct{ v int }{0}),
		{(func())(nil), (func())(nil)},
		ab(struct{ V any }{0}),
		ab((map[int]int)(nil)),
		{map[int]int{0: 0}, map[int]int{0: 0}},
		ab(map[int]float64{0: NaN}),
		ab(new(int)),
		{ptr(1), ptr(1)},
		ab(ptr(NaN)),
		ab([]int(nil)),
		{[]int{}, []int{}},
		{[]int{0}, []int{0}},
		ab([]float64{NaN}),
		ab(false),
		ab(int(0)),
		ab(int8(0)),
		ab(int16(0)),
		ab(int32(0)),
		ab(int64(0)),
		ab(uint(0)),
		ab(uint8(0)),
		ab(uint16(0)),
		ab(uint32(0)),
		ab(uint64(0)),
		ab(uintptr(0)),
		ab(float32(0)),
		ab(float64(0)),
		ab(complex64(0)),
		ab(complex128(0)),
		ab(""),
		ab(make(chan int)),
		ab(unsafe.Pointer(new(int))),
		ab(unsafe.Pointer(uintptr(0))),
	}

	for _, tt := range cases {
		t.Run(fmt.Sprintf("%v", tt), func(t *testing.T) {
			diff.Each(t.Errorf, tt[0], tt[1],
				diff.EqualFuncs(false),
			)
		})
		t.Run(fmt.Sprintf("unexported %v", tt), func(t *testing.T) {
			diff.Each(t.Errorf,
				struct{ v any }{tt[0]},
				struct{ v any }{tt[1]},
				diff.EqualFuncs(false),
			)
		})
	}
}

func TestUnequal(t *testing.T) {
	var cases = [][2]any{
		{[1]int{0}, [1]int{1}},
		{struct{ V int }{0}, struct{ V int }{1}},
		{(func())(nil), func() {}},
		{func() {}, func() {}},
		{struct{ V any }{0}, struct{ V any }{1}},
		{struct{ v any }{0}, struct{ v any }{1}},
		{(map[int]int)(nil), map[int]int{}},
		{(map[int]int)(nil), map[int]int{0: 0}},
		{(map[int]int)(nil), map[int]int{0: 0, 1: 1}},
		{map[int]int{}, map[int]int{0: 0}},
		{map[int]int{0: 0}, map[int]int{}},
		{map[int]int{0: 0}, map[int]int{0: 1}},
		{map[int]float64{0: NaN}, map[int]float64{0: NaN}},
		{nil, ptr(0)},
		{ptr(0), ptr(1)},
		{ptr(NaN), ptr(NaN)},
		{[]int(nil), []int{}},
		{[]int{}, []int(nil)},
		{[]int{0}, []int{1}},
		{[]float64{NaN}, []float64{NaN}},
		{false, true},
		{int(0), int(1)},
		{int8(0), int8(1)},
		{int16(0), int16(1)},
		{int32(0), int32(1)},
		{int64(0), int64(1)},
		{uint(0), uint(1)},
		{uint8(0), uint8(1)},
		{uint16(0), uint16(1)},
		{uint32(0), uint32(1)},
		{uint64(0), uint64(1)},
		{uintptr(0), uintptr(1)},
		{float32(0), float32(1)},
		{float64(0), float64(1)},
		{complex64(0), complex64(1)},
		{complex128(0), complex128(1)},
		{"", "a"},
		{make(chan int), make(chan int)},
		{unsafe.Pointer(ptr(0)), unsafe.Pointer(ptr(0))},
	}

	for i, tt := range cases {
		t.Run(fmt.Sprintf("%d: %v", i, tt), func(t *testing.T) {
			testUnequal(t, tt[0], tt[1])
		})
		t.Run(fmt.Sprintf("%d: unexported %v", i, tt), func(t *testing.T) {
			testUnequal(t,
				struct{ v any }{tt[0]},
				struct{ v any }{tt[1]},
			)
		})
	}
}

func TestCycle(t *testing.T) {
	type T struct {
		N int
		P *T
	}

	t.Run("equal and even", func(t *testing.T) {
		a := &T{N: 1, P: nil}
		a.P = a
		b := &T{N: 1, P: nil}
		b.P = b
		diff.Each(t.Errorf, a, b)
	})

	t.Run("unequal and even", func(t *testing.T) {
		a := &T{N: 1, P: nil}
		a.P = a
		b := &T{N: 2, P: nil}
		b.P = b
		testUnequal(t, a, b)
	})

	t.Run("equal and uneven", func(t *testing.T) {
		a := &T{N: 1, P: nil}
		a.P = a
		b1 := &T{N: 1, P: nil}
		b2 := &T{N: 1, P: b1}
		b1.P = b2
		testUnequal(t, a, b1)
		testUnequal(t, b1, a)
	})

	t.Run("equal and uneven x3", func(t *testing.T) {
		a := &T{N: 1, P: nil}
		a.P = a
		b1 := &T{N: 1, P: nil}
		b2 := &T{N: 1, P: b1}
		b3 := &T{N: 1, P: b2}
		b1.P = b3
		testUnequal(t, a, b1)
		testUnequal(t, b1, a)
	})
}

func TestPicky(t *testing.T) {
	type T struct{ v struct{ n int } }
	var a, b T
	b.v.n = 1
	equal := true
	f := func(format string, arg ...any) {
		equal = false
		t.Logf(format, arg...)
	}
	diff.Each(f, a, b, diff.Picky)
	if equal {
		t.Fail()
	}
}

func testUnequal(t *testing.T, a, b any) {
	t.Helper()
	equal := true
	sink := func(format string, arg ...any) {
		t.Helper()
		equal = false
		t.Logf(format, arg...)
	}
	diff.Each(sink, a, b,
		diff.EqualFuncs(false),
	)
	if equal {
		t.Fail()
	}
}

func ptr[T any](v T) *T {
	return &v
}
