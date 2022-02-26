package diffseq

import (
	"context"

	"github.com/pkg/diff/edit"
	"github.com/pkg/diff/myers"
)

// An Edit represents a single item in an edit script,
// either insert, replace, or delete.
// It contains only changed items, no surrounding equal context.
type Edit struct {
	A0, A1 int // range A[A0:A1]
	B0, B1 int // range B[B0:B1]
}

// merge transforms a edit.Script into a more useful edit script
// consisting of Edit values.
// the difference is that a edit.Script:
//   - represents a replacement as a delete plus an insert
//   - contains an item for each unchanged region
// which we don't want.
func merge(script edit.Script) (es []Edit) {
	needNext := true
	for _, r := range script.Ranges {
		switch r.Op() {
		case edit.Eq:
			needNext = true
		case edit.Del:
			if needNext {
				needNext = false
				es = append(es, Edit{
					A0: r.LowA, A1: r.HighA,
					B0: r.LowB, B1: r.HighB,
				})
			} else {
				es[len(es)-1].A1 = r.HighA
			}
		case edit.Ins:
			if needNext {
				needNext = false
				es = append(es, Edit{
					A0: r.LowA, A1: r.HighA,
					B0: r.LowB, B1: r.HighB,
				})
			} else {
				es[len(es)-1].B1 = r.HighB
			}
		}
	}
	return es
}

// A Seq represents a sequence of items to be compared
// against another sequence.
type Seq interface {
	Len() int
}

// Equal is a comparison function. It returns whether
// item ai in sequence a is equal to item bi in
// sequence b. It is okay for a and b to be the same
// sequence.
type Equal[S Seq] func(a, b S, ai, bi int) bool

// Diff finds an edit script to transform a into b.
// Function eq is used to determine equality of items.
func Diff[S Seq](a, b S, eq Equal[S]) []Edit {
	ctx := context.Background()
	return merge(myers.Diff(ctx, &pair[S]{a, b, eq}))
}

type pair[S Seq] struct {
	a, b S
	eq   Equal[S]
}

func (p *pair[S]) LenA() int { return p.a.Len() }
func (p *pair[S]) LenB() int { return p.b.Len() }
func (p *pair[S]) Equal(ai, bi int) bool {
	return p.eq(p.a, p.b, ai, bi)
}

// DiffSlice finds an edit script to transform a into b,
// using Go's built-in == operator.
func DiffSlice[T comparable](a, b []T) []Edit {
	return Diff[slice[T]](a, b, slice[T].ItemEq)
}

type slice[T comparable] []T

func (s slice[T]) Len() int { return len(s) }

func (a slice[T]) ItemEq(b slice[T], ai, bi int) bool {
	return a[ai] == b[bi]
}
