package diff_test

import (
	"testing"
	"time"

	"kr.dev/diff"
)

func TestTimeLocationEqual(t *testing.T) {
	tim := time.Now()
	t0 := tim.In(time.UTC)
	t1 := tim.In(time.Local)
	diff.Test(t, t.Errorf, t0, t1,
		diff.TimeEqual)

}

func TestTimeMonotonicEqual(t *testing.T) {
	tim := time.Now()
	t0 := tim.Add(5 * time.Millisecond)
	t1 := t0.Round(0)
	if !t0.Equal(t1) {
		t.Fatalf("oops, %v.Equal(%v) = false, want true (equal)", t0, t1)
	}
	if t0 == t1 {
		t.Fatalf("oops, %v == %v, want false (unequal)", t0, t1)
	}
	diff.Test(t, t.Errorf, t0, t1,
		diff.TimeEqual,
	)
}

func TestTimeUnequal(t *testing.T) {
	t0 := time.Now().In(time.UTC)
	t1 := t0.Add(5 * time.Millisecond)

	equal := true
	sink := func(format string, arg ...any) {
		t.Helper()
		equal = false
		t.Logf(format, arg...)
	}
	diff.Test(t, sink, t0, t1,
		diff.TimeEqual)

	if equal {
		t.Fail()
	}

}

func TestNoTimeLoc(t *testing.T) {
	tim := time.Now()
	t0 := tim.In(time.UTC)
	t1 := tim.In(time.Local)

	equal := true
	sink := func(format string, arg ...any) {
		t.Helper()
		equal = false
		t.Logf(format, arg...)
	}
	diff.Test(t, sink, t0, t1,
		diff.TransformRemove[time.Time]())

	if equal {
		t.Fail()
	}
}
