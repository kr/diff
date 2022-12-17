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
	"kr.dev/diff/internal/diffseq"
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
func Each(f func(format string, arg ...any) (int, error), a, b any, opt ...Option) {
	fdis := func(format string, arg ...any) { f(format, arg...) }
	var c config
	c.init(func() {}, fdis, opt...)
	each(a, b, &c)
}

// Log compares values a and b, printing each difference to its logger.
// By default, its logger object is log.Default()
// and its conditions for equality are like reflect.DeepEqual.
//
// The logger can be set using the Logger option.
// The behavior can also be adjusted by supplying other Option values.
// See Default for a complete list of default options.
// Values in opt apply in addition to (and override) the defaults.
func Log(a, b any, opt ...Option) {
	depth := stackDepth()
	var c config
	f := func(format string, arg ...any) {
		d := stackDepth() - depth
		c.output.Output(d+2, fmt.Sprintf(format, arg...))
	}
	c.init(func() {}, f, opt...)
	each(a, b, &c)
}

// Test compares values got and want, calling f for each difference it finds.
// By default, its conditions for equality are like reflect.DeepEqual.
//
// Test also calls h.Helper() at the top of every internal function.
// Note that *testing.T and *testing.B satisfy this interface.
// This makes test output show the file and line number of the call to
// Test.
//
// The behavior can be adjusted by supplying Option values.
// See Default for a complete list of default options.
// Values in opt apply in addition to (and override) the defaults.
func Test(h Helperer, f func(format string, arg ...any), got, want any, opt ...Option) {
	h.Helper()
	var c config
	c.init(h.Helper, f, opt...)
	c.inTest = true
	c.aLabel = "got"
	c.bLabel = "want"
	each(got, want, &c)
}

// Helperer marks the caller as a helper function.
// It is satisfied by *testing.T and *testing.B.
type Helperer interface {
	Helper()
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
	xform    map[reflect.Type]reflect.Value
	showOrig bool // also diff untransformed values

	format map[reflect.Type]reflect.Value

	helper func()
	output Outputter

	inTest bool
	aLabel string
	bLabel string
}

func (c *config) init(h func(), f func(format string, arg ...any), opt ...Option) {
	c.sink = f
	c.helper = h
	c.xform = map[reflect.Type]reflect.Value{}
	c.format = map[reflect.Type]reflect.Value{}
	c.aLabel = "a"
	c.bLabel = "b"
	defaultOpt.apply(c)
	OptionList(opt...).apply(c)
}

type visit struct {
	p unsafe.Pointer
	t reflect.Type
}

type emitter struct {
	config   config // not pointer, emitters have different configs
	rootType string
	path     []string
	av, bv   reflect.Value

	aSeen map[visit]visit
	bSeen map[visit]visit
}

func (e *emitter) set(av, bv reflect.Value) {
	e.av = av
	e.bv = bv
}

func (e *emitter) emitf(format string, arg ...any) {
	e.config.helper()
	switch e.config.level {
	case auto:
		var p string
		if len(e.path) > 0 {
			p = strings.Join(e.path, "") + ": "
		}
		arg = append([]any{e.rootType, p}, arg...)
		if strings.HasPrefix(format, "\n") && p == "" {
			format = format[1:]
		}
		e.config.sink("%s%s"+format+"\n", arg...)
	case pathOnly:
		e.config.sink("%s%s\n", e.rootType, strings.Join(e.path, ""))
	case full:
		var t string
		if e.rootType != "" {
			t = e.rootType + ":\n"
		} else if e.config.inTest {
			t = "any:\n"
		}
		p := strings.Join(e.path, "")
		e.config.sink("%s%s%s:\n%#v\n%s%s:\n%#v\n", t,
			e.config.aLabel, p, formatFull(e.av),
			e.config.bLabel, p, formatFull(e.bv),
		)
	default:
		panic("diff: bad verbose level")
	}
}

func (e *emitter) subf(t reflect.Type, format string, arg ...any) *emitter {
	if e.rootType == "" {
		var buf bytes.Buffer
		writeType(&buf, t)
		e.rootType = buf.String()
	}
	return &emitter{
		config:   e.config,
		rootType: e.rootType,
		path:     append(e.path, fmt.Sprintf(format, arg...)),
		aSeen:    e.aSeen,
		bSeen:    e.bSeen,
	}
}

func reflectApply(f reflect.Value, v ...reflect.Value) reflect.Value {
	return f.Call(v)[0]
}

func each(a, b any, c *config) {
	c.helper()
	e := &emitter{
		config: *c,
		aSeen:  map[visit]visit{},
		bSeen:  map[visit]visit{},
	}
	av := addressable(reflect.ValueOf(a))
	bv := addressable(reflect.ValueOf(b))
	walk(e, av, bv, true, true)
}

func equal(av, bv reflect.Value, c *config, xformOk bool) bool {
	var n int
	e := &emitter{
		config: *c,
		aSeen:  map[visit]visit{},
		bSeen:  map[visit]visit{},
	}
	e.config.format = nil
	e.config.sink = func(string, ...any) { n++ }
	walk(e, av, bv, xformOk, true)
	return n == 0
}

func walk(e *emitter, av, bv reflect.Value, xformOk, wantType bool) {
	e.config.helper()
	e.set(av, bv)
	if !av.IsValid() && !bv.IsValid() {
		return
	}
	if !av.IsValid() || !bv.IsValid() {
		e.emitf("%v != %v", formatShort(av, true), formatShort(bv, true))
		return
	}

	t := av.Type()
	if t != bv.Type() {
		e.emitf("%v != %v", formatShort(av, true), formatShort(bv, true))
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
		if bSeen, ok := e.aSeen[avis]; ok {
			if bSeen != bvis {
				e.emitf("uneven cycle")
			}
			return
		}
		if _, ok := e.bSeen[bvis]; ok {
			e.emitf("uneven cycle")
			return
		}
		e.aSeen[avis] = bvis
		e.bSeen[bvis] = avis
	}

	// Check for a transform func.
	if xf, haveXform := e.config.xform[t]; xformOk && haveXform {
		ax := addressable(reflectApply(xf, av).Elem())
		bx := addressable(reflectApply(xf, bv).Elem())
		walk(e.subf(t, "(transformed)"), ax, bx, false, true)
		if !e.config.showOrig {
			return
		}
		e = e.subf(t, "(original)")
		if equal(av, bv, &e.config, false) {
			e.emitf("equal")
			return
		}
	}

	// Check for a format func.
	if ff, ok := e.config.format[t]; ok {
		if !equal(av, bv, &e.config, false) {
			s := reflectApply(ff, av, bv).String()
			e.emitf("%s", s)
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
		seqDiff(e, av, bv)
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			afield := access(av.Field(i))
			bfield := access(bv.Field(i))
			walk(e.subf(t, "."+t.Field(i).Name), afield, bfield, true, false)
		}
	case reflect.Func:
		if e.config.equalFuncs {
			break
		}
		if !av.IsNil() || !bv.IsNil() {
			emitPointers(e, av, bv, wantType)
		}
	case reflect.Interface:
		aelem := addressable(av.Elem())
		belem := addressable(bv.Elem())
		walk(e, aelem, belem, xformOk, true)
	case reflect.Map:
		if av.IsNil() != bv.IsNil() {
			emitPointers(e, av, bv, wantType)
			break
		}
		if av.Pointer() == bv.Pointer() {
			break
		}

		for _, k := range sortedKeys(av, bv) {
			esub := e.subf(t, "[%#v]", k)
			ak := addressable(av.MapIndex(k))
			bk := addressable(bv.MapIndex(k))
			esub.set(ak, bk)
			if ak.IsValid() && bk.IsValid() {
				walk(esub, ak, bk, true, false)
			} else if ak.IsValid() {
				esub.emitf("(removed)")
			} else { // k in bv
				esub.emitf("(added) %v", formatShort(bk, false))
			}
		}
	case reflect.Ptr:
		if av.Pointer() == bv.Pointer() {
			break
		}
		if av.IsNil() != bv.IsNil() {
			e.emitf("%v != %v", formatShort(av, wantType), formatShort(bv, wantType))
			break
		}
		walk(e, av.Elem(), bv.Elem(), true, wantType)
	case reflect.Slice:
		if av.IsNil() != bv.IsNil() {
			emitPointers(e, av, bv, wantType)
			break
		}
		if av.Len() == bv.Len() && av.Pointer() == bv.Pointer() {
			break
		}
		if t.ConvertibleTo(reflectBytes) {
			as := av.Convert(reflectString)
			bs := bv.Convert(reflectString)
			stringDiff(e, t, as.String(), bs.String())
			break
		}
		seqDiff(e, av, bv)
	case reflect.Bool:
		eqtest(e, av, bv, av.Bool(), bv.Bool(), wantType)
	case reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64:
		eqtest(e, av, bv, av.Int(), bv.Int(), wantType)
	case reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		eqtest(e, av, bv, av.Uint(), bv.Uint(), wantType)
	case reflect.Float32, reflect.Float64:
		eqtest(e, av, bv, av.Float(), bv.Float(), wantType)
	case reflect.Complex64, reflect.Complex128:
		eqtest(e, av, bv, av.Complex(), bv.Complex(), wantType)
	case reflect.String:
		stringDiff(e, t, av.String(), bv.String())
	case reflect.Chan, reflect.UnsafePointer:
		if a, b := av.Pointer(), bv.Pointer(); a != b {
			emitPointers(e, av, bv, wantType)
		}
	default:
		panic("diff: unknown reflect.Kind " + t.Kind().String())
	}
}

func eqtest(e *emitter, av, bv reflect.Value, a, b any, wantType bool) {
	e.config.helper()
	if a != b {
		e.emitf("%v != %v",
			formatShort(av, wantType),
			formatShort(bv, wantType),
		)
	}
}

func emitPointers(e *emitter, av, bv reflect.Value, wantType bool) {
	e.config.helper()
	e.emitf("%v != %v",
		formatShort(av, wantType),
		formatShort(bv, wantType),
	)
}

func stringDiff(e *emitter, t reflect.Type, a, b string) {
	e.config.helper()

	if a == b {
		return
	}

	if utf8.ValidString(a) && utf8.ValidString(b) {
		textDiff(e, t, a, b)
		return
	}

	// TODO(kr): binary diff, hex, something
	e.emitf("binary: %+q != %+q", a, b)
}

func seqDiff(e *emitter, as, bs reflect.Value) {
	e.config.helper()
	eq := func(a, b reflect.Value, ai, bi int) bool {
		av := a.Index(ai)
		bv := b.Index(bi)
		return equal(av, bv, &e.config, true)
	}
	for _, ed := range diffseq.Diff(as, bs, eq) {
		a0, a1 := ed.A0, ed.A1
		b0, b1 := ed.B0, ed.B1
		if n := a1 - a0; n == b1-b0 {
			for i := 0; i < n; i++ {
				walk(e.subf(as.Type(), "[%d]", a0+i), as.Index(a0+i), bs.Index(b0+i), true, false)
			}
			continue
		}
		ee := e.subf(as.Type(), "[%d:%d]", a0, a1)
		afmt := formatShort(as.Slice(a0, a1), false)
		bfmt := formatShort(bs.Slice(b0, b1), false)
		ee.emitf("%v != %v", afmt, bfmt)
	}
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
