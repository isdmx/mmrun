package progress

import "io"

// Reader wraps an io.Reader and reports bytes read to a Bar.
type Reader struct {
	reader io.Reader
	bar    *Bar
}

// NewReader returns a Reader that calls bar.Add(n) on each Read.
func NewReader(r io.Reader, bar *Bar) *Reader {
	return &Reader{reader: r, bar: bar}
}

func (r *Reader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		r.bar.Add(int64(n))
	}
	return n, err
}
