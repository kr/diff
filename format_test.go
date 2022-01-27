package diff_test

import (
	"fmt"
	"testing"
	"time"

	"kr.dev/diff"
)

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
		got = fmt.Sprintf(format, arg...)
	}
	diff.Each(sink, t0, t1,
		diff.TimeDelta,
		diff.IgnoreUnexported(false),
	)
	if got != want {
		t.Fatalf("diff = %q, want %q", got, want)
	}
}
