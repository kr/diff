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
	writeValue(fs, f.v, true, true, 1)
}

func writeValue(w io.Writer, v reflect.Value, wantType, short bool, allowDepth int) {
	// TODO(kr): detect recursion during full output (short=false)
	if !v.IsValid() {
		io.WriteString(w, "nil") // untyped nil
		return
	}
	switch t := v.Type(); t.Kind() {
	case reflect.Array:
		if wantType {
			writeType(w, t)
		}
		if allowDepth < 1 && t.Len() > 0 {
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
			writeValue(w, v.Index(i), false, short, allowDepth-1)
		}
		io.WriteString(w, "}")
	case reflect.Struct:
		if wantType {
			writeType(w, t)
		}
		if allowDepth < 1 && t.NumField() > 0 {
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
			writeValue(w, v.Field(i), false, short, allowDepth-1)
		}
		io.WriteString(w, "}")
	case reflect.Func:
		if v.IsNil() {
			writeTypedNil(w, t, wantType)
			break
		}
		fmt.Fprintf(w, "%v {...}", t)
	case reflect.Interface:
		writeValue(w, v.Elem(), true, short, allowDepth)
	case reflect.Map:
		if v.IsNil() {
			writeTypedNil(w, t, wantType)
			break
		}
		if wantType {
			writeType(w, t)
		}
		if allowDepth < 1 && v.Len() > 0 {
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
			writeValue(w, mk, false, short, 0)
			io.WriteString(w, ":")
			writeValue(w, mv, false, short, allowDepth-1)
		}
		io.WriteString(w, "}")
	case reflect.Ptr:
		if v.IsNil() {
			writeTypedNil(w, t, wantType)
			break
		}
		if wantType || t.Elem().Kind() != reflect.Struct {
			io.WriteString(w, "&")
		}
		if t.Elem().Kind() == reflect.Pointer {
			// Two or more pointers in a row is confusing,
			// so show the type to be extra explicit.
			wantType = true
		}
		writeValue(w, v.Elem(), wantType, short, allowDepth) // note: don't decrement allowDepth
	case reflect.Slice:
		if v.IsNil() {
			writeTypedNil(w, t, wantType)
			break
		}
		if wantType {
			writeType(w, t)
		}
		if allowDepth < 1 && v.Len() > 0 {
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
			writeValue(w, v.Index(i), false, short, allowDepth-1)
		}
		io.WriteString(w, "}")
	case reflect.Bool:
		writeSimple(w, "%v", v, wantType && t.PkgPath() != "")
	case reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64:
		writeSimple(w, "%v", v, wantType)
	case reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		writeSimple(w, "%v", v, wantType)
	case reflect.Float32, reflect.Float64:
		writeSimple(w, "%v", v, wantType)
	case reflect.Complex64, reflect.Complex128:
		writeSimple(w, "%v", v, wantType)
	case reflect.String:
		// TODO(kr): abbreviate
		writeSimple(w, "%q", v, wantType && t.PkgPath() != "")
	case reflect.Chan:
		if v.IsNil() {
			writeTypedNil(w, t, wantType)
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

func writeSimple(w io.Writer, verb string, v reflect.Value, showType bool) {
	if showType {
		writeType(w, v.Type())
		io.WriteString(w, "(")
	}
	fmt.Fprintf(w, verb, v)
	if showType {
		io.WriteString(w, ")")
	}
}

func writeTypedNil(w io.Writer, t reflect.Type, showType bool) {
	// TODO(kr): print type name here sometimes (depending on context)
	if showType {
		needParens := false
		switch t.Kind() {
		case reflect.Func, reflect.Pointer, reflect.Chan:
			needParens = t.Name() == ""
		}
		if needParens {
			io.WriteString(w, "(")
		}
		writeType(w, t)
		if needParens {
			io.WriteString(w, ")")
		}
		io.WriteString(w, "(")
	}
	io.WriteString(w, "nil")
	if showType {
		io.WriteString(w, ")")
	}
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
		if t.NumField() > 0 {
			io.WriteString(w, " ")
		}
		for i := 0; i < t.NumField(); i++ {
			if i > 0 {
				io.WriteString(w, "; ")
			}
			field := t.Field(i)
			io.WriteString(w, field.Name)
			io.WriteString(w, " ")
			writeType(w, field.Type)
		}
		if t.NumField() > 0 {
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
