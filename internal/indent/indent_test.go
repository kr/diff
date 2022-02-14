package indent

import (
	"bytes"
	"testing"
)

func TestIndentPart(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf, []byte("aaa"))
	n, err := w.Write([]byte("bbb"))
	if n != 3 {
		t.Errorf("n = %d, want 3", n)
	}
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	got := buf.String()
	want := "aaabbb"
	if got != want {
		t.Errorf("got = %+q, want %+q", got, want)
	}
}

func TestIndentFull(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf, []byte("aaa"))
	n, err := w.Write([]byte("bbb\n"))
	if n != 4 {
		t.Errorf("n = %d, want 4", n)
	}
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	got := buf.String()
	want := "aaabbb\n"
	if got != want {
		t.Errorf("got = %+q, want %+q", got, want)
	}
}

func TestIndentMultiPart(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf, []byte("aaa"))
	n, err := w.Write([]byte("bbb\nccc"))
	if n != 7 {
		t.Errorf("n = %d, want 7", n)
	}
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	got := buf.String()
	want := "aaabbb\naaaccc"
	if got != want {
		t.Errorf("got = %+q, want %+q", got, want)
	}
}

func TestIndentMultiFull(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf, []byte("aaa"))
	n, err := w.Write([]byte("bbb\nccc\n"))
	if n != 8 {
		t.Errorf("n = %d, want 8", n)
	}
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	got := buf.String()
	want := "aaabbb\naaaccc\n"
	if got != want {
		t.Errorf("got = %+q, want %+q", got, want)
	}
}

func TestIndentMultiCallPart(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf, []byte("aaa"))
	n, err := w.Write([]byte("bb"))
	if n != 2 {
		t.Errorf("n = %d, want 2", n)
	}
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	n, err = w.Write([]byte("b\nccc"))
	if n != 5 {
		t.Errorf("n = %d, want 5", n)
	}
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	got := buf.String()
	want := "aaabbb\naaaccc"
	if got != want {
		t.Errorf("got = %+q, want %+q", got, want)
	}
}

func TestIndentMultiCallFull(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf, []byte("aaa"))
	n, err := w.Write([]byte("bbb\n"))
	if n != 4 {
		t.Errorf("n = %d, want 4", n)
	}
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	n, err = w.Write([]byte("ccc\n"))
	if n != 4 {
		t.Errorf("n = %d, want 4", n)
	}
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	got := buf.String()
	want := "aaabbb\naaaccc\n"
	if got != want {
		t.Errorf("got = %+q, want %+q", got, want)
	}
}
