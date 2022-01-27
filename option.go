package diff

import (
	"fmt"
	"math"
	"reflect"
	"time"
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
		IgnoreUnexported(true),
		TimeInUTC,
		TimeDelta,
	)
	defaultOpt = Default // actual value that cannot be changed

	// Picky is a set of options for exact comparison and
	// maximum verbosity.
	Picky Option = OptionList(
		EmitFull,
		EqualFuncs(false),
		IgnoreUnexported(false),
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
	// at that position.
	// It does not use registered format functions.
	EmitFull Option = verbosity(full)
)

var (
	// TimeInUTC converts Time values to UTC before comparison.
	// This effectively ignores the location, as in method Time.Equal.
	TimeInUTC Option = Transform(func(t time.Time) any {
		return t.In(time.UTC)
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

// IgnoreUnexported controls whether unexported fields will be compared.
// If true, unexported fields are skipped.
func IgnoreUnexported(b bool) Option {
	return Option{func(c *config) {
		c.ignoreUnexported = b
	}}
}

// EqualFuncs controls how function values are compared.
// If true, any two non-nil function values of the same type
// are treated as equal;
// otherwise, two non-nil functions are treated as unequal,
// even if they point to the same location in code.
// EqualFuncs(false) matches the behavior of the built-in == operator.
func EqualFuncs(b bool) Option {
	return Option{func(c *config) {
		c.equalFuncs = b
	}}
}

// Transform converts each value of type T to another value
// for the purpose of determining equality.
// The transformed value need not be the same type as T.
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
