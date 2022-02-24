package diff

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/pkg/diff/myers"
)

const nContext = 3

func (d *differ) textDiff(e emitfer, a, b string) {
	d.config.helper()

	// TODO(kr): check for whitespace-only changes, use special format

	if d.config.level == full {
		e.emitf("")
		return
	}

	// Check for multi-line.
	if textCheck(a, "\n", 2, 72) && textCheck(b, "\n", 2, 72) {
		e.emitf("\n%s", &diffTextFormatter{a, b, d.config.aLabel, d.config.bLabel})
		return
	}

	// Check for short strings.
	if len(a) < 20 && len(b) < 20 || a == "" || b == "" {
		e.emitf("%+q != %+q", a, b)
		return
	}

	// Check for multi-word.
	if textCheck(a, " ", 3, 10) && textCheck(b, " ", 3, 10) {
		as := strings.SplitAfter(a, " ")
		bs := strings.SplitAfter(b, " ")
		textDiffInline(e, a, b, as, bs)
		return
	}

	// Last resort is rune-by-rune.
	as := splitRunes(a)
	bs := splitRunes(b)
	textDiffInline(e, a, b, as, bs)
}

func textDiffInline(e emitfer, a, b string, as, bs []string) {
	acut := accum(as)
	bcut := accum(bs)
	pair := &slicePair[string]{a: as, b: bs}
	for _, ed := range merge(myers.Diff(context.Background(), pair)) {
		a0, a1 := acut[ed.a0], acut[ed.a1]
		b0, b1 := bcut[ed.b0], bcut[ed.b1]
		ee := e.subf(reflectString, "[%d:%d]", a0, a1)
		ee.emitf("%+q != %+q", a[a0:a1], b[b0:b1])
	}
}

func textCheck(s, sep string, nmin, amax int) bool {
	n := strings.Count(s, sep) + 1
	return n >= nmin && len(s)/n <= amax
}

type diffTextFormatter struct{ a, b, aLabel, bLabel string }

func (df *diffTextFormatter) Format(f fmt.State, verb rune) {
	fmt.Fprintf(f, "--- %s\n", df.aLabel)
	fmt.Fprintf(f, "+++ %s\n", df.bLabel)
	as := strings.Split(df.a, "\n")
	bs := strings.Split(df.b, "\n")
	ab := &slicePair[string]{a: as, b: bs}

	merged := merge(myers.Diff(context.Background(), ab))

	for i := 0; i < len(merged); {
		ed := merged[i]
		i1 := i + 1
		for i1 < len(merged) && (aIsClose(merged, i1) || bIsClose(merged, i1)) {
			i1++
		}
		ed1 := merged[i1-1]

		a0, b0 := 0, 0
		a1, b1 := len(as), len(bs)
		if n := ed.a0 - nContext; n > 0 {
			a0 = n
		}
		if n := ed.b0 - nContext; n > 0 {
			b0 = n
		}
		if n := ed1.a1 + nContext; n < a1 {
			a1 = n
		}
		if n := ed1.b1 + nContext; n < b1 {
			b1 = n
		}

		fmt.Fprintf(f, "@@ -%s +%s @@\n",
			lineRange(a0, a1-a0),
			lineRange(b0, b1-b0),
		)
		for a0 < a1 || b0 < b1 {
			if a0 < ed.a0 || i > i1 {
				fmt.Fprintf(f, " %s\n", as[a0])
				a0++
				b0++
			} else if a0 < ed.a1 {
				fmt.Fprintf(f, "-%s\n", as[a0])
				a0++
			} else if b0 < ed.b1 {
				fmt.Fprintf(f, "+%s\n", bs[b0])
				b0++
			}
			if a0 >= ed.a1 && b0 >= ed.b1 {
				i++
				if i < len(merged) {
					ed = merged[i]
				}
			}
		}
	}
}

func aIsClose(e []edit, i int) bool { return e[i].a0-e[i-1].a1 <= 2*nContext }
func bIsClose(e []edit, i int) bool { return e[i].b0-e[i-1].b1 <= 2*nContext }

func lineRange(r0, r1 int) string {
	switch r1 - r0 {
	case 0:
		return fmt.Sprintf("%d,0", r0)
	case 1:
		return strconv.Itoa(r0)
	}
	return fmt.Sprintf("%d,%d", r0+1, r1-r0)
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
