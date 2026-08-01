package progress

import (
	"io"
	"net/http"
)

// Transport wraps an http.RoundTripper and reports bytes transferred to a Bar.
type Transport struct {
	base http.RoundTripper
	bar  *Bar
}

// NewTransport creates a Transport. If base is nil, http.DefaultTransport is used.
func NewTransport(base http.RoundTripper, bar *Bar) *Transport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &Transport{base: base, bar: bar}
}

// RoundTrip implements http.RoundTripper.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil && req.Body != http.NoBody {
		req.Body = &countingBody{body: req.Body, bar: t.bar}
	}
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	if resp.Body != nil {
		resp.Body = &countingBody{body: resp.Body, bar: t.bar}
	}
	return resp, nil
}

type countingBody struct {
	body io.ReadCloser
	bar  *Bar
}

func (c *countingBody) Read(p []byte) (int, error) {
	n, err := c.body.Read(p)
	if n > 0 {
		c.bar.Add(int64(n))
	}
	return n, err
}

func (c *countingBody) Close() error { return c.body.Close() }
