package diff_test

import (
	"fmt"
	"math"
	"testing"

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
			}
			diff.Each(f, tt.a, tt.b, tt.opt)
			if got != tt.wantDiff {
				t.Errorf("diff = %v, want %v", got, tt.wantDiff)
			}
		})
	}
}
