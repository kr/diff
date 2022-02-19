package diff

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/pkg/diff"
	"github.com/pkg/diff/myers"
)

func (d *differ) textDiff(e emitfer, av, bv reflect.Value, a, b string) {
	d.config.helper()

	// TODO(kr): check for whitespace-only changes, use special format

	if d.config.level == full {
		e.emitf(av, bv, "")
		return
	}

	// Check for multi-line.
	if textCheck(a, "\n", 2, 72) && textCheck(b, "\n", 2, 72) {
		e.emitf(av, bv, "%s", &diffTextFormatter{a, b})
		return
	}

	// Check for short strings.
	if len(a) < 20 && len(b) < 20 || a == "" || b == "" {
		e.emitf(av, bv, "%+q != %+q", a, b)
		return
	}

	// Check for multi-word.
	if textCheck(a, " ", 3, 10) && textCheck(b, " ", 3, 10) {
		as := strings.SplitAfter(a, " ")
		bs := strings.SplitAfter(b, " ")
		textDiffInline(e, av, bv, a, b, as, bs)
		return
	}

	// Last resort is rune-by-rune.
	as := splitRunes(a)
	bs := splitRunes(b)
	textDiffInline(e, av, bv, a, b, as, bs)
}

func textDiffInline(e emitfer, av, bv reflect.Value, a, b string, as, bs []string) {
	acut := accum(as)
	bcut := accum(bs)
	pair := &slicePair[string]{a: as, b: bs}
	for _, ed := range merge(myers.Diff(context.Background(), pair)) {
		a0, a1 := acut[ed.a0], acut[ed.a1]
		b0, b1 := bcut[ed.b0], bcut[ed.b1]
		ee := e.subf(reflectString, "[%d:%d]", a0, a1)
		ee.emitf(av, bv, "%+q != %+q", a[a0:a1], b[b0:b1])
	}
}

func textCheck(s, sep string, nmin, amax int) bool {
	n := strings.Count(s, sep) + 1
	return n >= nmin && len(s)/n <= amax
}

type diffTextFormatter struct{ a, b string }

func (df *diffTextFormatter) Format(f fmt.State, verb rune) {
	err := diff.Text("a", "b", df.a, df.b, f)
	if err != nil {
		panic(err)
	}
}

type slicePair[T comparable] struct{ a, b []T }

func (ab *slicePair[T]) LenA() int             { return len(ab.a) }
func (ab *slicePair[T]) LenB() int             { return len(ab.b) }
func (ab *slicePair[T]) Equal(ai, bi int) bool { return ab.a[ai] == ab.b[bi] }

func accum(a []string) (is []int) {
	n, is := 0, append(is, 0)
	for _, sub := range a {
		n += len(sub)
		is = append(is, n)
	}
	return is
}

func splitRunes(s string) (a []string) {
	for s != "" {
		r, n := utf8.DecodeRuneInString(s)
		s = s[n:]
		a = append(a, string(r))
	}
	return a
}
