package diff_test

import (
	"testing"
	"time"

	"kr.dev/diff"
)

func TestTimeInUTCEqual(t *testing.T) {
	tim := time.Now()
	t0 := tim.In(time.UTC)
	t1 := tim.In(time.Local)
	diff.Each(t.Errorf, t0, t1,
		diff.TimeInUTC,
		diff.IgnoreUnexported(false),
	)
}

func TestTimeInUTCUnequal(t *testing.T) {
	t0 := time.Now().In(time.UTC)
	t1 := t0.Add(5 * time.Millisecond)

	equal := true
	sink := func(format string, arg ...any) {
		t.Helper()
		equal = false
		t.Logf(format, arg...)
	}
	diff.Each(sink, t0, t1,
		diff.TimeInUTC,
		diff.IgnoreUnexported(false),
	)
	if equal {
		t.Fail()
	}

}

func TestNoTimeInUTCLoc(t *testing.T) {
	tim := time.Now()
	t0 := tim.In(time.UTC)
	t1 := tim.In(time.Local)

	equal := true
	sink := func(format string, arg ...any) {
		t.Helper()
		equal = false
		t.Logf(format, arg...)
	}
	diff.Each(sink, t0, t1,
		diff.TransformRemove[time.Time](),
		diff.IgnoreUnexported(false),
	)
	if equal {
		t.Fail()
	}
}
