package diff_test

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"kr.dev/diff"
)

func TestEqualNaN(t *testing.T) {
	cases := []struct {
		opt      diff.Option
		a, b     float64
		wantDiff bool
	}{
		{diff.OptionList(), math.NaN(), math.NaN(), true},
		{diff.EqualNaN, math.NaN(), math.NaN(), false},
		{diff.EqualNaN, 1.0, 1.0, false},
		{diff.EqualNaN, 1.0, math.NaN(), true},
	}

	for _, tt := range cases {
		t.Run(fmt.Sprint(tt), func(t *testing.T) {
			got := false
			f := func(format string, arg ...any) {
				got = true
				t.Logf(format, arg...)
			}
			diff.Test(t, f, tt.a, tt.b, tt.opt)
			if got != tt.wantDiff {
				t.Errorf("diff = %v, want %v", got, tt.wantDiff)
			}
		})
	}
}

func TestTimeFormat(t *testing.T) {
	t0, err := time.Parse(time.RFC3339, "2021-01-31T12:39:00Z")
	if err != nil {
		t.Fatal(err)
	}
	t1 := t0.Add(5 * time.Millisecond)

	want := "2021-01-31T12:39:00Z != 2021-01-31T12:39:00.005Z (5ms)"
	var got string
	sink := func(format string, arg ...any) {
		t.Helper()
		t.Logf(format, arg...)
		got = strings.TrimSpace(fmt.Sprintf(format, arg...))
	}
	diff.Test(t, sink, t0, t1,
		diff.TimeDelta)

	if got != want {
		t.Fatalf("diff = %q, want %q", got, want)
	}
}

func TestZeroFields(t *testing.T) {
	type T struct{ A, B int }
	t0 := T{0, 2}
	t1 := T{1, 2}

	t.Run("A", func(t *testing.T) {
		diff.Test(t, t.Errorf, t0, t1,
			diff.ZeroFields[T]("A"))
	})

	t.Run("B", func(t *testing.T) {
		want := "diff_test.T.A: 0 != 1"
		var got string
		sink := func(format string, arg ...any) {
			t.Helper()
			t.Logf(format, arg...)
			got = strings.TrimSpace(fmt.Sprintf(format, arg...))
		}
		diff.Test(t, sink, t0, t1,
			diff.ZeroFields[T]("B"))
		if got != want {
			t.Fatalf("diff = %q, want %q", got, want)
		}
	})
}
