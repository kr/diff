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

// ZeroFields transforms a value of struct type T. It makes a copy of its input
// and sets the specified fields to their zero values.
//
// This effectively makes comparison ignore the given fields.
//
// See also Transform.
func ZeroFields[T any](fields ...string) Option {
	checkFieldsExist[T](fields...)
	return Transform(func(v T) any {
		e := reflect.ValueOf(&v).Elem()
		for _, name := range fields {
			fv := e.FieldByName(name)
			fv.Set(reflect.Zero(fv.Type()))
		}
		return v
	})
}

// KeepFields transforms a value of struct type T. It makes a copy of its input
// preserving the specified field values and setting all other fields to their
// zero values.
//
// This effectively makes comparison use only the provided fields.
//
// See also Transform.
func KeepFields[T any](fields ...string) Option {
	checkFieldsExist[T](fields...)
	return Transform(func(v0 T) any {
		var v1 T
		e0 := reflect.ValueOf(&v0).Elem()
		e1 := reflect.ValueOf(&v1).Elem()
		for _, name := range fields {
			if slices.Contains(fields, name) {
				fv0 := e0.FieldByName(name)
				fv1 := e1.FieldByName(name)
				fv1.Set(fv0)
			}
		}
		return v1
	})
}

func checkFieldsExist[T any](fields ...string) {
	t := reflect.TypeOf((*T)(nil)).Elem()
	for _, name := range fields {
		if _, ok := t.FieldByName(name); !ok {
			panic("diff: field not found: " + name)
		}
	}
}

// Transform converts each value of type T to another value
// for the purpose of determining equality.
// The transformed value need not be the same type as T.
//
// Function f must be pure. It must not incorporate
// randomness or rely on global state.
//
// A transform affects comparison, not output.
// The original, untransformed value is still emitted
// when a difference is found.
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
