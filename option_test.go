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

	want := "time.Time(transformed): 2021-01-31T12:39:00Z != 2021-01-31T12:39:00.005Z (5ms)"
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
	type C struct{ A, B int }
	t0 := C{0, 2}
	t1 := C{1, 2}

	t.Run("A", func(t *testing.T) {
		diff.Test(t, t.Errorf, t0, t1,
			diff.ZeroFields[C]("A"))
	})

	t.Run("B", func(t *testing.T) {
		want := "diff_test.C(transformed).A: 0 != 1"
		var got string
		sink := func(format string, arg ...any) {
			t.Helper()
			t.Logf(format, arg...)
			got = strings.TrimSpace(fmt.Sprintf(format, arg...))
		}
		diff.Test(t, sink, t0, t1,
			diff.ZeroFields[C]("B"))
		if got != want {
			t.Fatalf("diff = %q, want %q", got, want)
		}
	})
}

func TestKeepFields(t *testing.T) {
	type C struct{ A, B int }
	t0 := C{1, 2}
	t1 := C{1, 3}

	t.Run("A", func(t *testing.T) {
		diff.Test(t, t.Errorf, t0, t1,
			diff.KeepFields[C]("A"))
	})

	t.Run("B", func(t *testing.T) {
		want := "diff_test.C(transformed).B: 2 != 3"
		var got string
		sink := func(format string, arg ...any) {
			t.Helper()
			t.Logf(format, arg...)
			got = strings.TrimSpace(fmt.Sprintf(format, arg...))
		}
		diff.Test(t, sink, t0, t1,
			diff.KeepFields[C]("B"))
		if got != want {
			t.Fatalf("diff = %q, want %q", got, want)
		}
	})
}

func TestKeepExported(t *testing.T) {
	type em struct{ I int }
	type C struct {
		em
		A, B, unexported int
	}
	t0 := C{em{3}, 1, 2, 9}
	t1 := C{em{4}, 1, 2, 5}
	diff.Test(t, t.Errorf, t0, t1, diff.KeepExported[C]())

	var e any
	func() {
		defer func() { e = recover() }()
		diff.KeepExported[struct{}]()
	}()
	if e == nil {
		t.Errorf("expected panic")
	}
}
