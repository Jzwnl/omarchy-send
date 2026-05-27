package client

import (
	"io"
	"time"
)

// progressReader wraps a file reader, emitting throttled byte-count callbacks as
// the body is streamed to the peer.
type progressReader struct {
	r        io.Reader
	total    int64
	read     int64
	emit     func(sent int64)
	lastEmit time.Time
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.read += int64(n)
	if err == io.EOF || time.Since(p.lastEmit) > 100*time.Millisecond {
		p.lastEmit = time.Now()
		if p.emit != nil {
			p.emit(p.read)
		}
	}
	return n, err
}
