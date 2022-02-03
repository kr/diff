package diff

import (
	"fmt"
	"reflect"
	"strings"
	"unsafe"
)

// Each compares values a and b, calling f for each difference it finds.
// By default, its conditions for equality are similar to reflect.DeepEqual.
//
// The behavior can be adjusted by supplying Option values.
// See Default for a complete list of default options.
// Values in opt apply in addition to (and override) the defaults.
func Each(f func(format string, arg ...any), a, b any, opt ...Option) {
	d := &differ{
		aSeen: map[visit]visit{},
		bSeen: map[visit]visit{},
	}
	d.config.xform = map[reflect.Type]reflect.Value{}
	d.config.format = map[reflect.Type]reflect.Value{}
	OptionList(defaultOpt, OptionList(opt...)).apply(&d.config)
	e := &printEmitter{sink: f, level: d.config.level}
	d.walk(e, reflect.ValueOf(a), reflect.ValueOf(b), true)
}

type differ struct {
	config config
	aSeen  map[visit]visit
	bSeen  map[visit]visit
}

type config struct {
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
}

type visit struct {
	p unsafe.Pointer
	t reflect.Type
}

type emitfer interface {
	emitf(av, bv reflect.Value, format string, arg ...any)
	subf(format string, arg ...any) emitfer
	didEmit() bool
}

type printEmitter struct {
	level level
	path  []string
	did   bool
	sink  func(format string, a ...any)
}

func (e *printEmitter) emitf(av, bv reflect.Value, format string, arg ...any) {
	e.did = true
	var p string
	if len(e.path) > 0 {
		p = strings.Join(e.path, "") + ": "
	}
	switch e.level {
	case auto:
		arg = append([]any{p}, arg...)
		e.sink("%s"+format, arg...)
	case pathOnly:
		e.sink("%s", strings.Join(e.path, ""))
	case full:
		e.sink("%s%#v != %#v", p, av, bv)
	default:
		panic("diff: bad verbose level")
	}
}

func (e *printEmitter) subf(format string, arg ...any) emitfer {
	return &printEmitter{
		level: e.level,
		path:  append(e.path, fmt.Sprintf(format, arg...)),
		did:   false,
		sink: func(format string, a ...any) {
			e.did = true
			e.sink(format, a...)
		},
	}
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

func (e *countEmitter) subf(format string, arg ...any) emitfer {
	return e
}

func (e *countEmitter) didEmit() bool {
	return e.n > 0
}

func reflectApply(f reflect.Value, v ...reflect.Value) reflect.Value {
	return f.Call(v)[0]
}

func (d *differ) equal(av, bv reflect.Value) bool {
	d2 := &differ{
		config: d.config,
		aSeen:  map[visit]visit{},
		bSeen:  map[visit]visit{},
	}
	d2.config.xform = nil
	d2.config.format = nil
	e := &countEmitter{}
	d2.walk(e, av, bv, true)
	return !e.didEmit()
}

func (d *differ) walk(e emitfer, av, bv reflect.Value, xformOk bool) {
	if !av.IsValid() && !bv.IsValid() {
		return
	}
	if !av.IsValid() {
		e.emitf(av, bv, "nil != %v", bv.Type())
		return
	}
	if !bv.IsValid() {
		e.emitf(av, bv, "%v != nil", av.Type())
		return
	}

	t := av.Type()
	if bt := bv.Type(); t != bt {
		e.emitf(av, bv, "%v != %v", t, bt)
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
	var ax, bx reflect.Value
	var haveXform bool
	if xformOk {
		var xf reflect.Value
		xf, haveXform = d.config.xform[t]
		if haveXform && xformOk {
			ax = reflectApply(xf, av)
			bx = reflectApply(xf, bv)
			if d.equal(ax, bx) {
				return
			}
		}
	}

	// Check for a format func.
	if ff, ok := d.config.format[t]; ok && !d.equal(av, bv) {
		s := reflectApply(ff, av, bv).String()
		e.emitf(av, bv, "%s", s)
		return
	}

	// We use almost the same rules as reflect.DeepEqual here,
	// but with a couple of configuration options that modify
	// the behavior, such as:
	//   * We allow the client to ignore functions.
	//   * We allow the client to ignore unexported fields.
	// See "go doc reflect DeepEqual" for more.
	switch t.Kind() {
	case reflect.Array:
		// TODO(kr): fancy diff (histogram, myers)
		for i := 0; i < t.Len(); i++ {
			d.walk(e.subf("[%d]", i), av.Index(i), bv.Index(i), true)
		}
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			d.walk(e.subf("."+t.Field(i).Name), av.Field(i), bv.Field(i), true)
		}
	case reflect.Func:
		if d.config.equalFuncs {
			break
		}
		if !av.IsNil() || !bv.IsNil() {
			emitPointers(e, av, bv)
		}
	case reflect.Interface:
		d.walk(e, av.Elem(), bv.Elem(), true)
	case reflect.Map:
		if av.IsNil() != bv.IsNil() {
			emitPointers(e, av, bv)
			break
		}
		if av.Pointer() == bv.Pointer() {
			break
		}
		ak, both, bk := keyDiff(av, bv)
		for _, k := range ak {
			e.subf("[%#v]", k).
				emitf(av.MapIndex(k), bv.MapIndex(k), "(removed)")
		}
		for _, k := range both {
			d.walk(e.subf("[%#v]", k), av.MapIndex(k), bv.MapIndex(k), true)
		}
		for _, k := range bk {
			e.subf("[%#v]", k).
				emitf(av.MapIndex(k), bv.MapIndex(k), "(added) %#v", bv.MapIndex(k))
		}
	case reflect.Ptr:
		if av.Pointer() == bv.Pointer() {
			break
		}
		d.walk(e, av.Elem(), bv.Elem(), true)
	case reflect.Slice:
		if av.IsNil() != bv.IsNil() {
			emitPointers(e, av, bv)
			break
		}
		if av.Len() == bv.Len() && av.Pointer() == bv.Pointer() {
			break
		}
		// TODO(kr): fancy diff (histogram, myers)
		n := av.Len()
		if blen := bv.Len(); n != blen {
			e.emitf(av, bv, "{len %d} != {len %d}", n, blen)
			return
		}
		for i := 0; i < n; i++ {
			d.walk(e.subf("[%d]", i), av.Index(i), bv.Index(i), true)
		}
	case reflect.Bool:
		eqtest(e, av, bv, av.Bool(), bv.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64:
		eqtest(e, av, bv, av.Int(), bv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		eqtest(e, av, bv, av.Uint(), bv.Uint())
	case reflect.Float32, reflect.Float64:
		eqtest(e, av, bv, av.Float(), bv.Float())
	case reflect.Complex64, reflect.Complex128:
		eqtest(e, av, bv, av.Complex(), bv.Complex())
	case reflect.String:
		if a, b := av.String(), bv.String(); a != b {
			e.emitf(av, bv, "%q != %q", a, b)
		}
	case reflect.Chan, reflect.UnsafePointer:
		if a, b := av.Pointer(), bv.Pointer(); a != b {
			emitPointers(e, av, bv)
		}
	default:
		panic("diff: unknown reflect.Kind " + t.Kind().String())
	}

	// The xform check returns early if the transformed values are
	// deeply equal. So if we got this far, we know they are different.
	// If we didn't find a difference in the untransformed values, make
	// sure to emit *something*, and then diff the *transformed* values.
	if haveXform && !e.didEmit() {
		e.emitf(av, bv, "(transformed values differ)")
		d.walk(e.subf("->"), ax, bx, false)
	}
}

func eqtest[V comparable](e emitfer, av, bv reflect.Value, a, b V) {
	if a != b {
		e.emitf(av, bv, "%v != %v", a, b)
	}
}

func emitPointers(e emitfer, av, bv reflect.Value) {
	ap := unsafe.Pointer(av.Pointer())
	bp := unsafe.Pointer(bv.Pointer())
	if av.IsNil() {
		e.emitf(av, bv, "nil != %p", bp)
	} else if bv.IsNil() {
		e.emitf(av, bv, "%p != nil", ap)
	} else {
		e.emitf(av, bv, "%p != %p", ap, bp)
	}
}

func keyDiff(av, bv reflect.Value) (ak, both, bk []reflect.Value) {
	for aIter := av.MapRange(); aIter.Next(); {
		k := aIter.Key()
		if !bv.MapIndex(k).IsValid() {
			ak = append(ak, k)
		} else {
			both = append(both, k)
		}
	}
	for bIter := bv.MapRange(); bIter.Next(); {
		k := bIter.Key()
		if !av.MapIndex(k).IsValid() {
			bk = append(bk, k)
		}
	}
	return ak, both, bk
}
