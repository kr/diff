package diff

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
	"unsafe"
)

func TestWriteShortUnknownContext(t *testing.T) {
	var any0 any = 0
	var any1 any = 1
	cases := []struct {
		a, b any
		want string
	}{
		{nil, 1, `nil != int(1)`},
		{[1]int{0}, [1]int{1}, `[1]int[0]: 0 != 1`},
		{[1]any{0}, [1]any{1}, `[1]any[0]: int(0) != int(1)`},
		{struct{ v int }{0}, struct{ v int }{1}, `struct{ v int }.v: 0 != 1`},
		{struct{ v any }{0}, struct{ v any }{1}, `struct{ v any }.v: int(0) != int(1)`},
		{map[int]int{1: 0}, map[int]int{1: 1}, `map[int]int[1]: 0 != 1`},
		{map[int]any{1: 0}, map[int]any{1: 1}, `map[int]any[1]: int(0) != int(1)`},
		{map[int]int{}, map[int]int{1: 1}, `map[int]int[1]: (added) 1`},
		{map[int]any{}, map[int]any{1: 1}, `map[int]any[1]: (added) int(1)`},
		{[1]*int{nil}, [1]*int{ptr(1)}, `[1]*int[0]: nil != &1`},
		{[1]*int{ptr(0)}, [1]*int{ptr(1)}, `[1]*int[0]: 0 != 1`},
		{[1]*any{nil}, [1]*any{&any1}, `[1]*any[0]: nil != &int(1)`},
		{[1]*any{&any0}, [1]*any{&any1}, `[1]*any[0]: int(0) != int(1)`},
		{[]int{0}, []int{1}, `[]int[0]: 0 != 1`},
		{[]any{0}, []any{1}, `[]any[0]: int(0) != int(1)`},
		{[]any{(*int)(nil)}, []any{ptr(1)}, `[]any[0]: (*int)(nil) != &int(1)`},
	}

	for i, tt := range cases {
		t.Run(fmt.Sprint(i, ":", tt), func(t *testing.T) {
			got := ""
			sink := func(format string, arg ...any) {
				t.Helper()
				got = strings.TrimSpace(fmt.Sprintf(format, arg...))
			}
			Test(t, sink, tt.a, tt.b)
			t.Logf("got: %s", got)
			if got != tt.want {
				t.Errorf("Test(%#v, %#v) = %#q, want %#q", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestWriteShortSpecial(t *testing.T) {
	// This test is for values that can't be hard coded because
	// they are liable to change every time.
	cases := []struct {
		v    any
		want []string
	}{
		{make(chan int), []string{"(chan int)(0x", ")"}},
		{unsafe.Pointer(new(int)), []string{"unsafe.Pointer(0x", ")"}},

		// Elide concrete type when it's known from context.
		//  * nilable types
		{[1]chan int{make(chan int)}, []string{"[1]chan int{(chan int)(0x", ")}"}},

		// Show concrete type when it's not known from context.
		//  * nilable types
		{[1]any{make(chan int)}, []string{"[1]any{(chan int)(0x", ")}"}},
	}

	for i, tt := range cases {
		t.Run(fmt.Sprint(i, ":", tt), func(t *testing.T) {
			rv := reflect.ValueOf(tt.v)
			got := fmt.Sprint(formatShort(rv, true))
			t.Logf("got: %s", got)
			for _, want := range tt.want {
				i := strings.Index(got, want)
				if i < 0 {
					t.Fatalf("formatShort(%#v) remaining: %#q, want %#q", tt.v, got, want)
				}
				got = got[i+len(want):]
			}
		})
	}
}

func TestWriteShort(t *testing.T) {
	type Bool bool
	type Int int
	type String string
	type Empty struct{}
	type Slice []int
	type Chan chan int
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
		{(func())(nil), "(func())(nil)"},
		{func() {}, "func() {...}"},
		{map[int]int(nil), "map[int]int(nil)"},
		{map[int]int{}, "map[int]int{}"},
		{map[int]int{0: 0}, "map[int]int{0:0}"},
		{map[int]int{0: 0, 1: 1, 2: 2, 3: 3, 4: 4}, "map[int]int{0:0, ...}"},
		{(*int)(nil), "(*int)(nil)"},
		{ptr(0), "&int(0)"},
		{ptr(ptr(0)), "&&int(0)"},
		{&T{V: 0}, "&diff.T{V:int(0)}"},
		{[]int(nil), "[]int(nil)"},
		{[]int{}, "[]int{}"},
		{[]int{0}, "[]int{0}"},
		{[]int{0, 1}, "[]int{0, ...}"},
		{false, "false"},
		{0, "int(0)"},
		{"a", `"a"`},
		{(chan int)(nil), "(chan int)(nil)"},
		{Chan(nil), "diff.Chan(nil)"},
		{unsafe.Pointer(nil), "unsafe.Pointer(0x0)"},

		// Truncate nested values.
		{T{V: [1]int{0}}, "diff.T{V:[1]int{...}}"},
		{[]any{[0]int{}}, "[]any{[0]int{}}"},
		{T{V: T{V: 0}}, "diff.T{V:diff.T{...}}"},
		{T{V: map[int]int{0: 0}}, "diff.T{V:map[int]int{...}}"},
		{T{V: map[int]int{}}, "diff.T{V:map[int]int{}}"},
		{T{V: []int{0}}, "diff.T{V:[]int{...}}"},
		{T{V: []int{}}, "diff.T{V:[]int{}}"},
		{[1]any{T{V: 0}}, "[1]any{diff.T{...}}"},
		{map[int]any{0: T{V: 0}}, "map[int]any{0:diff.T{...}}"},
		{[]any{T{V: 0}}, "[]any{diff.T{...}}"},
		{[]any{Empty{}}, "[]any{diff.Empty{}}"},

		// Elide concrete type when it's known from context.
		//  * non-nilable containers
		{[1][0]int{{}}, "[1][0]int{{}}"},
		{[1]struct{}{{}}, "[1]struct{}{{}}"},
		{[1]Empty{{}}, "[1]diff.Empty{{}}"},

		// Elide concrete type when it's known from context.
		//  * nilable types
		{[1]func(){func() {}}, "[1]func(){func() {...}}"},
		{[1]map[int]int{{}}, "[1]map[int]int{{}}"},
		{[1]*int{ptr(1)}, "[1]*int{&1}"},
		{[1]**int{ptr(ptr(1))}, "[1]**int{&&int(1)}"},                  // 2+ ptrs, don't elide
		{[1]**int{ptr((*int)(nil))}, "[1]**int{&(*int)(nil)}"},         // 2+ ptrs, don't elide
		{[1]**Empty{ptr(&Empty{})}, "[1]**diff.Empty{&&diff.Empty{}}"}, // 2+ ptrs, don't elide
		{[1][]int{{}}, "[1][]int{{}}"},
		{[1]Slice{{}}, "[1]diff.Slice{{}}"},

		// Elide concrete type when it's known from context.
		//  * nil
		{[1]func(){nil}, "[1]func(){nil}"},
		{[1]map[int]int{nil}, "[1]map[int]int{nil}"},
		{[1]*int{nil}, "[1]*int{nil}"},
		{[1]**int{nil}, "[1]**int{nil}"},
		{[1][]int{nil}, "[1][]int{nil}"},
		{[1]chan int{nil}, "[1]chan int{nil}"},

		// Elide concrete type when it's known from context.
		//  * simple types
		{[1]bool{false}, "[1]bool{false}"},
		{[1]Bool{false}, "[1]diff.Bool{false}"},
		{[1]int{0}, "[1]int{0}"},
		{[1]Int{0}, "[1]diff.Int{0}"},
		{[1]string{"a"}, `[1]string{"a"}`},
		{[1]String{"a"}, `[1]diff.String{"a"}`},

		// Show concrete type when it's not known from context.
		//  * non-nilable containers
		{[1]any{[0]int{}}, "[1]any{[0]int{}}"},
		{[1]any{struct{}{}}, "[1]any{struct{}{}}"},
		{[1]any{Empty{}}, "[1]any{diff.Empty{}}"},

		// Show concrete type when it's not known from context.
		//  * nilable types
		{[1]any{func() {}}, "[1]any{func() {...}}"},
		{[1]any{map[int]int{}}, "[1]any{map[int]int{}}"},
		{[1]any{ptr(1)}, "[1]any{&int(1)}"},
		{[1]any{ptr(ptr(1))}, "[1]any{&&int(1)}"},
		{[1]any{ptr((*int)(nil))}, "[1]any{&(*int)(nil)}"},
		{[1]any{[]int{}}, "[1]any{[]int{}}"},
		{[1]any{Slice{}}, "[1]any{diff.Slice{}}"},

		// Show concrete type when it's not known from context.
		//  * nil
		{[1]any{(func())(nil)}, "[1]any{(func())(nil)}"},
		{[1]any{map[int]int(nil)}, "[1]any{map[int]int(nil)}"},
		{[1]any{(*int)(nil)}, "[1]any{(*int)(nil)}"},
		{[1]any{(**int)(nil)}, "[1]any{(**int)(nil)}"},
		{[1]any{[]int(nil)}, "[1]any{[]int(nil)}"},
		{[1]any{(chan int)(nil)}, "[1]any{(chan int)(nil)}"},

		// Show concrete type when it's not known from context.
		//  * simple types
		{[1]any{false}, "[1]any{false}"},
		{[1]any{Bool(false)}, "[1]any{diff.Bool(false)}"},
		{[1]any{0}, "[1]any{int(0)}"},
		{[1]any{Int(0)}, "[1]any{diff.Int(0)}"},
		{[1]any{"a"}, `[1]any{"a"}`},
		{[1]any{String("a")}, `[1]any{diff.String("a")}`},

		// Special case, omit "&" during elision inside arrays & slices.
		{[1]*Empty{{}}, "[1]*diff.Empty{{}}"},
		{[]*Empty{{}}, "[]*diff.Empty{{}}"},
		{[1]any{&Empty{}}, "[1]any{&diff.Empty{}}"},
		{[]any{&Empty{}}, "[]any{&diff.Empty{}}"},

		// Trigger elision from each container.
		{[1]int{0}, "[1]int{0}"},
		{struct{ V int }{0}, "struct{ V int }{V:0}"},
		{map[int]int{0: 0}, "map[int]int{0:0}"},
		{[]int{0}, "[]int{0}"},

		// Trigger non-elision from each container.
		{[1]any{0}, "[1]any{int(0)}"},
		{struct{ V any }{0}, "struct{ V any }{V:int(0)}"},
		{map[int]any{0: 0}, "map[int]any{0:int(0)}"},
		{[]any{0}, "[]any{int(0)}"},
	}

	for i, tt := range cases {
		t.Run(fmt.Sprint(i, ":", tt), func(t *testing.T) {
			rv := reflect.ValueOf(tt.v)
			got := fmt.Sprint(formatShort(rv, true))
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
