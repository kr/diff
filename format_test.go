package diff

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"testing"
	"unsafe"
)

func TestWriteShort(t *testing.T) {
	type T struct{ V any }
	cases := []struct {
		v    any
		want string
	}{
		{nil, "nil"},
		{[1]int{0}, "[1]int{0}"},
		{[2]int{}, "[2]int{0, ...}"},
		{struct{ V int }{0}, "struct{ V int }{V:0}"},
		{struct{ V, U int }{}, "struct{ V int; U int }{V:0, ...}"},
		{struct{ V int }{0}, "struct{ V int }{V:0}"},
		{(func())(nil), "nil"},
		{func() {}, "func() {...}"},
		{(map[int]int)(nil), "nil"},
		{map[int]int{}, "map[int]int{}"},
		{map[int]int{0: 0}, "map[int]int{0:0}"},
		{map[int]int{0: 0, 1: 1}, "map[int]int{0:0, ...}"},
		{(*int)(nil), "nil"},
		{ptr(0), "&0"},
		{ptr(ptr(0)), "&&0"},
		{([]int)(nil), "nil"},
		{[]int{}, "[]int{}"},
		{[]int{0}, "[]int{0}"},
		{[]int{0, 1}, "[]int{0, ...}"},
		{false, "false"},
		{0, "0"},
		{"a", `"a"`},
		{(chan int)(nil), "nil"},
		{make(chan int), "chan int"},
		{unsafe.Pointer(new(int)), "unsafe.Pointer(...)"},
		{unsafe.Pointer(uintptr(0)), "unsafe.Pointer(0)"},
		{T{V: [1]int{0}}, "diff.T{V:[1]int{...}}"},
		{T{V: T{V: 0}}, "diff.T{V:diff.T{...}}"},
		{T{V: map[int]int{0: 0}}, "diff.T{V:map[int]int{...}}"},
		{T{V: []int{0}}, "diff.T{V:[]int{...}}"},
		{[1]any{T{V: 0}}, "[1]any{diff.T{...}}"},
		{map[int]any{0: T{V: 0}}, "map[int]any{0:diff.T{...}}"},
		{[]any{T{V: 0}}, "[]any{diff.T{...}}"},
	}

	for i, tt := range cases {
		t.Run(fmt.Sprint(i, ":", tt), func(t *testing.T) {
			rv := reflect.ValueOf(tt.v)
			got := fmt.Sprint(formatShort(rv))
			t.Logf("got: %s", got)
			if got != tt.want {
				t.Errorf("formatShort(%#v) = %#q, want %#q", tt.v, got, tt.want)
			}
		})
	}
}

func TestWriteType(t *testing.T) {
	type T struct{}
	testWriteType[any](t, "any")
	testWriteType[[0]any](t, "[0]any")
	testWriteType[struct{}](t, "struct{}")
	testWriteType[struct{ V any }](t, "struct{ V any }")
	testWriteType[struct{ V, U any }](t, "struct{ V any; U any }")
	testWriteType[func()](t, "func()")
	testWriteType[func(any)](t, "func(any)")
	testWriteType[func(any, any)](t, "func(any, any)")
	testWriteType[func(int, ...bool)](t, "func(int, ...bool)")
	testWriteType[func() any](t, "func() any")
	testWriteType[func() (any, any)](t, "func() (any, any)")
	testWriteType[func(x any) (y any)](t, "func(any) any")
	testWriteType[interface{ F() }](t, "interface{ F() }")
	testWriteType[interface{ F(any) }](t, "interface{ F(any) }")
	testWriteType[interface{ F(any, any) }](t, "interface{ F(any, any) }")
	testWriteType[interface{ F() any }](t, "interface{ F() any }")
	testWriteType[interface{ F() (any, any) }](t, "interface{ F() (any, any) }")
	testWriteType[interface {
		F()
		G()
	}](t, "interface{ F(); G() }")
	testWriteType[map[any]any](t, "map[any]any")
	testWriteType[*any](t, "*any")
	testWriteType[[]any](t, "[]any")
	testWriteType[chan any](t, "chan any")
	testWriteType[chan<- any](t, "chan<- any")
	testWriteType[<-chan any](t, "<-chan any")
	testWriteType[bool](t, "bool")
	testWriteType[int](t, "int")
	testWriteType[string](t, "string")
	testWriteType[T](t, "diff.T")
	testWriteType[io.Reader](t, "io.Reader")
	testWriteType[unsafe.Pointer](t, "unsafe.Pointer")
}

func testWriteType[T any](t *testing.T, want string) {
	t.Helper()
	rt := reflect.TypeOf((*T)(nil)).Elem()
	var buf bytes.Buffer
	writeType(&buf, rt)
	got := buf.String()
	t.Logf("got: %s", got)
	if got != want {
		t.Errorf("writeType(%v) = %#q, want %#q", rt, got, want)
	}
}

func ptr[T any](v T) *T {
	return &v
}
