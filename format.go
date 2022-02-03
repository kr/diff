package diff

import (
	"fmt"
	"io"
	"reflect"
)

var reflectAny = reflect.TypeOf((*any)(nil)).Elem()

func formatShort(v reflect.Value) fmt.Formatter {
	return &formatter{
		v: v,
	}
}

type formatter struct {
	v reflect.Value
}

func (f *formatter) Format(fs fmt.State, verb rune) {
	writeValue(fs, f.v, true, 1)
}

func writeValue(w io.Writer, v reflect.Value, short bool, n int) {
	// TODO(kr): detect recursion during full output (short=false)
	if !v.IsValid() {
		io.WriteString(w, "nil") // untyped nil
		return
	}
	switch t := v.Type(); t.Kind() {
	case reflect.Array:
		writeType(w, t)
		if n < 1 {
			io.WriteString(w, "{...}")
			break
		}
		io.WriteString(w, "{")
		for i := 0; i < t.Len(); i++ {
			if i > 0 {
				io.WriteString(w, ", ")
				if short {
					io.WriteString(w, "...")
					break
				}
			}
			writeValue(w, v.Index(i), short, n-1)
		}
		io.WriteString(w, "}")
	case reflect.Struct:
		writeType(w, t)
		if n < 1 {
			io.WriteString(w, "{...}")
			break
		}
		io.WriteString(w, "{")
		for i := 0; i < t.NumField(); i++ {
			if i > 0 {
				io.WriteString(w, ", ")
				if short {
					io.WriteString(w, "...")
					break
				}
			}
			io.WriteString(w, t.Field(i).Name)
			io.WriteString(w, ":")
			writeValue(w, v.Field(i), short, n-1)
		}
		io.WriteString(w, "}")
	case reflect.Func:
		if v.IsNil() {
			writeTypedNil(w, t)
			break
		}
		fmt.Fprintf(w, "%v {...}", t)
	case reflect.Interface:
		writeValue(w, v.Elem(), short, n)
	case reflect.Map:
		if v.IsNil() {
			writeTypedNil(w, t)
			break
		}
		writeType(w, t)
		if n < 1 {
			io.WriteString(w, "{...}")
			break
		}
		io.WriteString(w, "{")
		first := true
		for it := v.MapRange(); it.Next(); first = false {
			if !first {
				io.WriteString(w, ", ")
				if short {
					io.WriteString(w, "...")
					break
				}
			}
			mk := it.Key()
			mv := v.MapIndex(mk)
			writeValue(w, mk, short, 0)
			io.WriteString(w, ":")
			writeValue(w, mv, short, n-1)
		}
		io.WriteString(w, "}")
	case reflect.Ptr:
		if v.IsNil() {
			writeTypedNil(w, t)
			break
		}
		io.WriteString(w, "&")
		writeValue(w, v.Elem(), short, n) // note: don't decrement n
	case reflect.Slice:
		if v.IsNil() {
			writeTypedNil(w, t)
			break
		}
		writeType(w, t)
		if n < 1 {
			io.WriteString(w, "{...}")
			break
		}
		io.WriteString(w, "{")
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				io.WriteString(w, ", ")
				if short {
					io.WriteString(w, "...")
					break
				}
			}
			writeValue(w, v.Index(i), short, n-1)
		}
		io.WriteString(w, "}")
	case reflect.Bool:
		fmt.Fprintf(w, "%v", v)
	case reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64:
		writeNumber(w, v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		writeNumber(w, v)
	case reflect.Float32, reflect.Float64:
		writeNumber(w, v)
	case reflect.Complex64, reflect.Complex128:
		writeNumber(w, v)
	case reflect.String:
		// TODO(kr): abbreviate
		fmt.Fprintf(w, "%q", v.String())
	case reflect.Chan:
		if v.IsNil() {
			writeTypedNil(w, t)
			break
		}
		writeType(w, t)
	case reflect.UnsafePointer:
		if v.Pointer() == 0 {
			io.WriteString(w, "unsafe.Pointer(0)")
		} else {
			io.WriteString(w, "unsafe.Pointer(...)")
		}
	default:
		w.Write([]byte("(unknown kind)"))
	}
}

func writeNumber(w io.Writer, v reflect.Value) {
	// TODO(kr): print type name here sometimes (depending on context)
	fmt.Fprintf(w, "%v", v)
}

func writeTypedNil(w io.Writer, t reflect.Type) {
	// TODO(kr): print type name here sometimes (depending on context)
	// needParens := t.Name() == ""
	io.WriteString(w, "nil")
}

func writeType(w io.Writer, t reflect.Type) {
	if t == reflectAny {
		io.WriteString(w, "any")
		return
	}

	if name := t.Name(); name != "" {
		io.WriteString(w, t.String())
		return
	}

	switch t.Kind() {
	case reflect.Array:
		fmt.Fprintf(w, "[%d]", t.Len())
		writeType(w, t.Elem())
	case reflect.Struct:
		io.WriteString(w, "struct{")
		n := t.NumField()
		if n > 0 {
			io.WriteString(w, " ")
		}
		for i := 0; i < n; i++ {
			if i > 0 {
				io.WriteString(w, "; ")
			}
			field := t.Field(i)
			io.WriteString(w, field.Name)
			io.WriteString(w, " ")
			writeType(w, field.Type)
		}
		if n > 0 {
			io.WriteString(w, " ")
		}
		io.WriteString(w, "}")
	case reflect.Func:
		io.WriteString(w, "func")
		writeFunc(w, t)
	case reflect.Interface:
		io.WriteString(w, "interface{ ")
		for i := 0; i < t.NumMethod(); i++ {
			if i > 0 {
				io.WriteString(w, "; ")
			}
			method := t.Method(i)
			io.WriteString(w, method.Name)
			writeFunc(w, method.Type)
		}
		io.WriteString(w, " }")
	case reflect.Map:
		io.WriteString(w, "map[")
		writeType(w, t.Key())
		io.WriteString(w, "]")
		writeType(w, t.Elem())
	case reflect.Ptr:
		io.WriteString(w, "*")
		writeType(w, t.Elem())
	case reflect.Slice:
		io.WriteString(w, "[]")
		writeType(w, t.Elem())
	case reflect.Chan:
		if t.ChanDir() == reflect.RecvDir {
			io.WriteString(w, "<-")
		}
		io.WriteString(w, "chan")
		if t.ChanDir() == reflect.SendDir {
			io.WriteString(w, "<-")
		}
		io.WriteString(w, " ")
		writeType(w, t.Elem())
	default:
		fmt.Fprint(w, t)
	}
}

func writeFunc(w io.Writer, f reflect.Type) {
	io.WriteString(w, "(")
	n := f.NumIn()
	for i := 0; i < n; i++ {
		if i > 0 {
			io.WriteString(w, ", ")
		}
		if i == n-1 && f.IsVariadic() {
			io.WriteString(w, "...")
			writeType(w, f.In(i).Elem())
		} else {
			writeType(w, f.In(i))
		}
	}
	io.WriteString(w, ")")
	n = f.NumOut()
	if n > 0 {
		io.WriteString(w, " ")
	}
	if n > 1 {
		io.WriteString(w, "(")
	}
	for i := 0; i < n; i++ {
		if i > 0 {
			io.WriteString(w, ", ")
		}
		writeType(w, f.Out(i))
	}
	if n > 1 {
		io.WriteString(w, ")")
	}
}
