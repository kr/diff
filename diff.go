package diff

import (
	"bytes"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"unicode/utf8"
	"unsafe"

	"github.com/rogpeppe/go-internal/fmtsort"
)

var (
	reflectBytes  = reflect.TypeOf((*[]byte)(nil)).Elem()
	reflectString = reflect.TypeOf((*string)(nil)).Elem()
	reflectBool   = reflect.TypeOf(true)
)

var (
	reflectTrue = reflect.ValueOf(true)
)

// Each compares values a and b, calling f for each difference it finds.
// By default, its conditions for equality are like reflect.DeepEqual.
//
// The behavior can be adjusted by supplying Option values.
// See Default for a complete list of default options.
// Values in opt apply in addition to (and override) the defaults.
func Each(f func(format string, arg ...any), a, b any, opt ...Option) {
	d := newDiffer(func() {}, f, opt...)
	d.each(a, b)
}

// Log compares values a and b, printing each difference to its logger.
// By default, its conditions for equality are like reflect.DeepEqual.
//
// Log provides a calldepth argument to its logger to show the file
// and line number of the call to Log. This is usually preferable to
// passing log.Printf to Each.
//
// The default logger object is log.Default().
// It can be set using the Logger option.
// The behavior can also be adjusted by supplying other Option values.
// See Default for a complete list of default options.
// Values in opt apply in addition to (and override) the defaults.
func Log(a, b any, opt ...Option) {
	depth := stackDepth()
	var d *differ
	f := func(format string, arg ...any) {
		dd := stackDepth() - depth
		d.config.output.Output(dd+2, fmt.Sprintf(format, arg...))
	}
	d = newDiffer(func() {}, f, opt...)
	d.each(a, b)
}

// Test compares values a and b, calling f for each difference it finds.
// By default, its conditions for equality are like reflect.DeepEqual.
//
//
// Test also calls h.Helper() at the top of every internal function.
// Note that *testing.T and *testing.B satisfy this interface.
// This makes test output show the file and line number of the call to
// Test.
//
// The behavior can be adjusted by supplying Option values.
// See Default for a complete list of default options.
// Values in opt apply in addition to (and override) the defaults.
func Test(h Helperer, f func(format string, arg ...any), a, b any, opt ...Option) {
	h.Helper()
	d := newDiffer(h.Helper, f, opt...)
	d.each(a, b)
}

// Helperer marks the caller as a helper function.
// It is satisfied by *testing.T and *testing.B.
type Helperer interface {
	Helper()
}

type differ struct {
	config config
	aSeen  map[visit]visit
	bSeen  map[visit]visit
}

type config struct {
	sink func(format string, a ...any)

	level level // verbosity

	// equalFuncs treats non-nil functions as equal.
	// In the == operator, non-nil function values
	// are never equal, so it is often useless to compare them.
	equalFuncs bool

	// xform transforms values of the given type before
	// they are included in the diff tree.
	// hashes, weights, and differences are computed
	// using the transformed values.
	xform map[reflect.Type]reflect.Value

	format map[reflect.Type]reflect.Value

	helper func()
	output Outputter
}

type visit struct {
	p unsafe.Pointer
	t reflect.Type
}

type emitfer interface {
	emitf(av, bv reflect.Value, format string, arg ...any)
	subf(t reflect.Type, format string, arg ...any) emitfer
	didEmit() bool
}

type printEmitter struct {
	config config // not pointer, printEmitters have different configs
	path   []string
	did    bool
}

func (e *printEmitter) emitf(av, bv reflect.Value, format string, arg ...any) {
	e.config.helper()
	e.did = true
	var p string
	if len(e.path) > 0 {
		p = strings.Join(e.path, "") + ": "
	}
	switch e.config.level {
	case auto:
		arg = append([]any{p}, arg...)
		e.config.sink("%s"+format+"\n", arg...)
	case pathOnly:
		e.config.sink("%s\n", strings.Join(e.path, ""))
	case full:
		e.config.sink("%s%#v != %#v\n", p, av, bv)
	default:
		panic("diff: bad verbose level")
	}
}

func (e *printEmitter) subf(t reflect.Type, format string, arg ...any) emitfer {
	path := e.path
	if len(e.path) < 1 {
		var buf bytes.Buffer
		writeType(&buf, t)
		path = []string{buf.String()}
	}
	pe := &printEmitter{
		config: e.config,
		path:   append(path, fmt.Sprintf(format, arg...)),
		did:    false,
	}
	pe.config.sink = func(format string, a ...any) {
		e.config.helper()
		e.did = true
		e.config.sink(format, a...)
	}
	return pe
}

func (e *printEmitter) didEmit() bool {
	return e.did
}

type countEmitter struct {
	n int
}

func (e *countEmitter) emitf(av, bv reflect.Value, format string, arg ...any) {
	e.n++
}

func (e *countEmitter) subf(t reflect.Type, format string, arg ...any) emitfer {
	return e
}

func (e *countEmitter) didEmit() bool {
	return e.n > 0
}

func reflectApply(f reflect.Value, v ...reflect.Value) reflect.Value {
	return f.Call(v)[0]
}

func newDiffer(h func(), f func(format string, arg ...any), opt ...Option) *differ {
	d := &differ{
		aSeen: map[visit]visit{},
		bSeen: map[visit]visit{},
	}
	d.config.sink = f
	d.config.helper = h
	d.config.xform = map[reflect.Type]reflect.Value{}
	d.config.format = map[reflect.Type]reflect.Value{}
	OptionList(defaultOpt, OptionList(opt...)).apply(&d.config)
	return d
}

func (d *differ) each(a, b any) {
	d.config.helper()
	e := &printEmitter{config: d.config}
	av := addressable(reflect.ValueOf(a))
	bv := addressable(reflect.ValueOf(b))
	d.walk(e, av, bv, true, true)
}

func (d *differ) equalAsIs(av, bv reflect.Value) bool {
	d2 := &differ{
		config: d.config,
		aSeen:  map[visit]visit{},
		bSeen:  map[visit]visit{},
	}
	d2.config.format = nil
	e := &countEmitter{}
	d2.walk(e, av, bv, false, true)
	return !e.didEmit()
}

func (d *differ) walk(e emitfer, av, bv reflect.Value, xformOk, wantType bool) {
	d.config.helper()
	if !av.IsValid() && !bv.IsValid() {
		return
	}
	if !av.IsValid() || !bv.IsValid() {
		e.emitf(av, bv, "%v != %v", formatShort(av, true), formatShort(bv, true))
		return
	}

	t := av.Type()
	if t != bv.Type() {
		e.emitf(av, bv, "%v != %v", formatShort(av, true), formatShort(bv, true))
		return
	}

	// Check for cycles.
	switch t.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice:
		if av.IsNil() || bv.IsNil() {
			break
		}
		avis := visit{unsafe.Pointer(av.Pointer()), t}
		bvis := visit{unsafe.Pointer(bv.Pointer()), t}
		if bSeen, ok := d.aSeen[avis]; ok {
			if bSeen != bvis {
				e.emitf(av, bv, "uneven cycle")
			}
			return
		}
		if _, ok := d.bSeen[bvis]; ok {
			e.emitf(av, bv, "uneven cycle")
			return
		}
		d.aSeen[avis] = bvis
		d.bSeen[bvis] = avis
	}

	// Check for a transform func.
	didXform := false
	if xf, haveXform := d.config.xform[t]; xformOk && haveXform {
		ax := addressable(reflectApply(xf, av).Elem())
		bx := addressable(reflectApply(xf, bv).Elem())
		if d.equalAsIs(ax, bx) {
			return
		}
		didXform = true
	}

	// Check for a format func.
	if ff, ok := d.config.format[t]; ok {
		if didXform || !d.equalAsIs(av, bv) {
			s := reflectApply(ff, av, bv).String()
			e.emitf(av, bv, "%s", s)
		}
		return
	}

	// We use almost the same rules as reflect.DeepEqual here,
	// but with a couple of configuration options that modify
	// the behavior, such as:
	//   * We allow the client to ignore functions.
	// See "go doc reflect DeepEqual" for more.
	switch t.Kind() {
	case reflect.Array:
		// TODO(kr): fancy diff (histogram, myers)
		for i := 0; i < t.Len(); i++ {
			d.walk(e.subf(t, "[%d]", i), av.Index(i), bv.Index(i), true, false)
		}
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			afield := access(av.Field(i))
			bfield := access(bv.Field(i))
			d.walk(e.subf(t, "."+t.Field(i).Name), afield, bfield, true, false)
		}
	case reflect.Func:
		if d.config.equalFuncs {
			break
		}
		if !av.IsNil() || !bv.IsNil() {
			d.emitPointers(e, av, bv, wantType)
		}
	case reflect.Interface:
		aelem := addressable(av.Elem())
		belem := addressable(bv.Elem())
		d.walk(e, aelem, belem, xformOk, true)
	case reflect.Map:
		if av.IsNil() != bv.IsNil() {
			d.emitPointers(e, av, bv, wantType)
			break
		}
		if av.Pointer() == bv.Pointer() {
			break
		}

		for _, k := range sortedKeys(av, bv) {
			if av.MapIndex(k).IsValid() && bv.MapIndex(k).IsValid() {
				d.walk(e.subf(t, "[%#v]", k), av.MapIndex(k), bv.MapIndex(k), true, false)
			} else if av.MapIndex(k).IsValid() {
				e.subf(t, "[%#v]", k).
					emitf(av.MapIndex(k), bv.MapIndex(k), "(removed)")
			} else { // k in bv
				e.subf(t, "[%#v]", k).
					emitf(av.MapIndex(k), bv.MapIndex(k), "(added) %v", formatShort(bv.MapIndex(k), false))
			}
		}
	case reflect.Ptr:
		if av.Pointer() == bv.Pointer() {
			break
		}
		if av.IsNil() != bv.IsNil() {
			e.emitf(av, bv, "%v != %v", formatShort(av, wantType), formatShort(bv, wantType))
			break
		}
		d.walk(e, av.Elem(), bv.Elem(), true, wantType)
	case reflect.Slice:
		if av.IsNil() != bv.IsNil() {
			d.emitPointers(e, av, bv, wantType)
			break
		}
		if av.Len() == bv.Len() && av.Pointer() == bv.Pointer() {
			break
		}
		if t.ConvertibleTo(reflectBytes) {
			as := av.Convert(reflectString)
			bs := bv.Convert(reflectString)
			stringDiff(e, av, bv, as.String(), bs.String())
			break
		}
		// TODO(kr): fancy diff (histogram, myers)
		n := av.Len()
		if blen := bv.Len(); n != blen {
			e.emitf(av, bv, "{len %d} != {len %d}", n, blen)
			return
		}
		for i := 0; i < n; i++ {
			d.walk(e.subf(t, "[%d]", i), av.Index(i), bv.Index(i), true, false)
		}
	case reflect.Bool:
		d.eqtest(e, av, bv, av.Bool(), bv.Bool(), wantType)
	case reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64:
		d.eqtest(e, av, bv, av.Int(), bv.Int(), wantType)
	case reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		d.eqtest(e, av, bv, av.Uint(), bv.Uint(), wantType)
	case reflect.Float32, reflect.Float64:
		d.eqtest(e, av, bv, av.Float(), bv.Float(), wantType)
	case reflect.Complex64, reflect.Complex128:
		d.eqtest(e, av, bv, av.Complex(), bv.Complex(), wantType)
	case reflect.String:
		stringDiff(e, av, bv, av.String(), bv.String())
	case reflect.Chan, reflect.UnsafePointer:
		if a, b := av.Pointer(), bv.Pointer(); a != b {
			d.emitPointers(e, av, bv, wantType)
		}
	default:
		panic("diff: unknown reflect.Kind " + t.Kind().String())
	}

	// The xform check returns early if the transformed values are
	// deeply equal. So if we got this far, we know they are different.
	// If we didn't find a difference in the untransformed values,
	// the xform function can't be a pure function.
	// Make sure to emit *something* so the user knows there is a diff.
	if didXform && !e.didEmit() {
		var buf bytes.Buffer
		writeType(&buf, t)
		e.emitf(av, bv, "warning: %s transform is impure", buf.String())
		e.emitf(av, bv, "%v != %v", formatShort(av, wantType), formatShort(bv, wantType))
	}
}

func (d *differ) eqtest(e emitfer, av, bv reflect.Value, a, b any, wantType bool) {
	d.config.helper()
	if a != b {
		e.emitf(av, bv, "%v != %v",
			formatShort(av, wantType),
			formatShort(bv, wantType),
		)
	}
}

func (d *differ) emitPointers(e emitfer, av, bv reflect.Value, wantType bool) {
	d.config.helper()
	e.emitf(av, bv, "%v != %v",
		formatShort(av, wantType),
		formatShort(bv, wantType),
	)
}

func stringDiff(e emitfer, av, bv reflect.Value, a, b string) {
	if a == b {
		return
	}

	u := utf8.ValidString(a) && utf8.ValidString(b)
	if !u {
		// TODO(kr): binary diff, hex, something
		e.emitf(av, bv, "binary: %+q != %+q", a, b)
		return
	}

	textDiff(e, av, bv, a, b)
}

func sortedKeys(maps ...reflect.Value) []reflect.Value {
	t := reflect.MapOf(maps[0].Type().Key(), reflectBool)
	merged := reflect.MakeMap(t)
	for _, m := range maps {
		iter := m.MapRange()
		for iter.Next() {
			merged.SetMapIndex(iter.Key(), reflectTrue)
		}
	}
	return fmtsort.Sort(merged).Key
}

func addressable(r reflect.Value) reflect.Value {
	if !r.IsValid() {
		return r
	}
	a := reflect.New(r.Type()).Elem()
	a.Set(r)
	return a
}

func access(v reflect.Value) reflect.Value {
	p := unsafe.Pointer(v.UnsafeAddr())
	return reflect.NewAt(v.Type(), p).Elem()
}

func stackDepth() int {
	pc := make([]uintptr, 1000)
	return runtime.Callers(0, pc)
}
