package diff

import diffedit "github.com/pkg/diff/edit"

// an edit represents a single item in an edit script,
// either insert, replace, or delete.
// it contains only changed items, no surrounding equal context.
type edit struct {
	a0, a1 int // range A[a0:a1]
	b0, b1 int // range B[b0:b1]
}

// merge transforms a diffedit.Script into a more useful edit script
// consisting of edit values.
// the difference is that a diffedit.Script:
//   - represents a replacement as a delete plus an insert
//   - contains an item for each unchanged region
// which we don't want.
func merge(script diffedit.Script) (es []edit) {
	needNext := true
	for _, r := range script.Ranges {
		switch r.Op() {
		case diffedit.Eq:
			needNext = true
		case diffedit.Del:
			if needNext {
				needNext = false
				es = append(es, edit{
					a0: r.LowA, a1: r.HighA,
					b0: r.LowB, b1: r.HighB,
				})
			} else {
				es[len(es)-1].a1 = r.HighA
			}
		case diffedit.Ins:
			if needNext {
				needNext = false
				es = append(es, edit{
					a0: r.LowA, a1: r.HighA,
					b0: r.LowB, b1: r.HighB,
				})
			} else {
				es[len(es)-1].b1 = r.HighB
			}
		}
	}
	return es
}
