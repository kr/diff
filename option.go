package diff

import (
	"fmt"
	"log"
	"math"
	"reflect"
	"time"

	"golang.org/x/exp/slices"
)

// A level describes how much output to produce.
type level int

const (
	auto level = iota
	pathOnly
	full
)

// Option values can be passed to the Each function to control
// how comparisons are made, how output is formatted,
// and various other things.
// Options are applied in order from left to right;
// later options win where there is a conflict.
type Option struct{ apply func(*config) }

// OptionList combines multiple options into one.
// The arguments will be applied in order from left to right.
func OptionList(opt ...Option) Option {
	return Option{func(c *config) {
		for _, o := range opt {
			o.apply(c)
		}
	}}
}

var (
	// Default is a copy of the default options used by Each.
	// (This variable is only for documentation;
	// modifying it has no effect on the default behavior.)
	Default Option = OptionList(
		EmitAuto,
		TimeEqual,
		TimeDelta,
		Logger(log.Default()),
	)
	defaultOpt = Default // actual value that cannot be changed

	// Picky is a set of options for exact comparison and
	// maximum verbosity.
	Picky Option = OptionList(
		EmitFull,
		TransformRemove[time.Time](),
		FormatRemove[time.Time](),
	)
)

var (
	// EmitAuto selects an output format for each difference
	// based on various heuristics.
	// It uses registered format functions. See Format.
	EmitAuto Option = verbosity(auto)

	// EmitPathOnly outputs the path to each difference
	// in Go notation.
	// It does not use registered format functions.
	EmitPathOnly Option = verbosity(pathOnly)

	// EmitFull outputs the path to each difference
	// and a full representation of both values
	// at that position, pretty-printed on multiple
	// lines with indentation.
	EmitFull Option = verbosity(full)
)

var (
	// TimeEqual converts Time values to a form that can be compared
	// meaningfully by the == operator.
	// See the documentation on the Time type and Time.Equal
	// for an explanation.
	TimeEqual Option = Transform(func(t time.Time) any {
		return t.Round(0).UTC()
	})

	// EqualNaN causes NaN float64 values to be treated as equal.
	EqualNaN Option = Transform(func(f float64) any {
		if math.IsNaN(f) {
			type equalNaN struct{}
			return equalNaN{}
		}
		return f
	})

	// TimeDelta outputs the difference between two times
	// in a more readable format, including the delta between them.
	TimeDelta Option = Format(func(a, b time.Time) string {
		as := a.Format(time.RFC3339Nano)
		bs := b.Format(time.RFC3339Nano)
		return fmt.Sprintf("%s != %s (%s)", as, bs, b.Sub(a))
	})
)

// verbosity controls how much detail is produced for each difference found.
func verbosity(n level) Option {
	return Option{func(c *config) {
		c.level = n
	}}
}

// ShowOriginal show diffs of untransformed values in addition
// to the diffs of transformed values. This is mainly useful for
// debugging transform functions.
func ShowOriginal() Option {
	return Option{func(c *config) {
		c.showOrig = true
	}}
}

// EqualFuncs controls how function values are compared.
// If true, any two non-nil function values of the same type
// are treated as equal;
// otherwise, two non-nil functions are treated as unequal,
// even if they point to the same location in code.
// Note that EqualFuncs(false) matches the behavior of the built-in == operator.
func EqualFuncs(b bool) Option {
	return Option{func(c *config) {
		c.equalFuncs = b
	}}
}

// ZeroFields transforms values of struct type T. It makes a copy of its input
// and sets the named fields to their zero values.
//
// This effectively makes comparison ignore the given fields.
//
// ZeroFields panics if any name argument is not a visible field in T.
// See Transform for more info about transforms.
// See also KeepFields.
func ZeroFields[T any](name ...string) Option {
	checkFieldsExist[T](name)
	return Transform(func(v T) any {
		e := reflect.ValueOf(&v).Elem()
		for _, s := range name {
			fv := e.FieldByName(s)
			fv.Set(reflect.Zero(fv.Type()))
		}
		return v
	})
}

// KeepFields transforms values of struct type T. It makes a copy of its input,
// preserving the named field values and setting all other fields to their
// zero values.
//
// This effectively makes comparison use only the provided fields.
//
// KeepFields panics if any name argument is not a visible field in T.
// See Transform for more info about transforms.
// See also ZeroFields.
func KeepFields[T any](name ...string) Option {
	checkFieldsExist[T](name)
	return Transform(func(v0 T) any {
		var v1 T
		e0 := reflect.ValueOf(&v0).Elem()
		e1 := reflect.ValueOf(&v1).Elem()
		for _, s := range name {
			if slices.Contains(name, s) {
				fv0 := e0.FieldByName(s)
				fv1 := e1.FieldByName(s)
				fv1.Set(fv0)
			}
		}
		return v1
	})
}

// KeepExported transforms a value of struct type T. It makes a copy of its input,
// preserving exported fields only.
//
// This effectively makes comparison use only exported fields.
//
// See also Transform.
func KeepExported[T any]() Option {
	t := reflect.TypeOf((*T)(nil)).Elem()
	var fields []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.IsExported() {
			fields = append(fields, f.Name)
		}
	}

	if len(fields) == 0 {
		panic("diff: struct must contain at least one exported field")
	}

	return KeepFields[T](fields...)
}

func checkFieldsExist[T any](fields []string) {
	t := reflect.TypeOf((*T)(nil)).Elem()
	for _, name := range fields {
		if _, ok := t.FieldByName(name); !ok {
			panic("diff: field not found: " + name)
		}
	}
}

// Transform converts values of type T to another value to
// be compared.
//
// A transform is applied when the values on both sides of
// the comparison are of type T. The two values returned by
// the transform (one on each side) are then compared, and
// their diffs emitted. See also ShowOriginal.
//
// Function f may return any type, not just T. In
// particular, during a single comparison, f may return a
// different type on each side (and this will result in a
// difference being reported).
//
// See TransformRemove to remove a transform.
func Transform[T any](f func(T) any) Option {
	return Option{func(c *config) {
		t := reflect.TypeOf((*T)(nil)).Elem()
		c.xform[t] = reflect.ValueOf(f)
	}}
}

// TransformRemove removes any transform for type T.
// See Transform.
func TransformRemove[T any]() Option {
	return Option{func(c *config) {
		t := reflect.TypeOf((*T)(nil)).Elem()
		delete(c.xform, t)
	}}
}

// Format customizes the description of the difference
// between two unequal values a and b.
//
// See FormatRemove to remove a custom format.
func Format[T any](f func(a, b T) string) Option {
	return Option{func(c *config) {
		t := reflect.TypeOf((*T)(nil)).Elem()
		c.format[t] = reflect.ValueOf(f)
	}}
}

// FormatRemove removes any format for type T.
// See Format.
func FormatRemove[T any]() Option {
	return Option{func(c *config) {
		t := reflect.TypeOf((*T)(nil)).Elem()
		delete(c.format, t)
	}}
}

// Outputter accepts log output.
// It is satisfied by *log.Logger.
type Outputter interface {
	Output(calldepth int, s string) error
}

// Logger sets the output for Log to the given object.
// It has no effect on Each or Test.
func Logger(out Outputter) Option {
	return Option{func(c *config) {
		c.output = out
	}}
}
