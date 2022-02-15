package indent

import (
	"bytes"
	"io"
)

type writer struct {
	w      io.Writer
	prefix string
	bol    bool
}

func New(w io.Writer, prefix string) io.Writer {
	return &writer{
		w:      w,
		prefix: prefix,
		bol:    true,
	}
}

func (w *writer) Write(p []byte) (written int, err error) {
	for len(p) > 0 {
		if w.bol {
			_, err := io.WriteString(w.w, w.prefix)
			w.bol = false
			if err != nil {
				return written, err
			}
		}
		i := bytes.IndexByte(p, '\n')
		if i < 0 {
			n, err := w.w.Write(p)
			written += n
			return written, err
		}
		n, err := w.w.Write(p[:i+1])
		written += n
		if err != nil {
			return written, err
		}
		p = p[i+1:]
		w.bol = true
	}
	return written, err
}
