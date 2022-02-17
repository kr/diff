package diff

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/diff"
)

func (d *differ) textDiff(e emitfer, av, bv reflect.Value, a, b string) {
	d.config.helper()

	// TODO(kr): check for whitespace-only changes, use special format

	// Check for short strings.
	if len(a) < 30 && len(b) < 30 {
		e.emitf(av, bv, "%+q != %+q", a, b)
		return
	}

	// Check for multi-line.
	if textCheck(a, "\n", 2, 72) && textCheck(b, "\n", 2, 72) {
		e.emitf(av, bv, "%s", &diffTextFormatter{a, b})
		return
	}

	// Check for multi-word.
	if textCheck(a, " ", 3, 10) && textCheck(b, " ", 3, 10) {
		// TODO(kr): don't show words on separate lines
		as := strings.SplitAfter(a, " ")
		bs := strings.SplitAfter(b, " ")
		e.emitf(av, bv, "%s", &diffSlicesFormatter{as, bs})
		return
	}

	// Last resort is byte-by-byte.
	// TODO(kr): inline results like multi-word? something
	e.emitf(av, bv, "%+q != %+q", a, b)
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

type diffSlicesFormatter struct{ a, b any }

func (df *diffSlicesFormatter) Format(f fmt.State, verb rune) {
	err := diff.Slices("a", "b", df.a, df.b, f)
	if err != nil {
		panic(err)
	}
}
