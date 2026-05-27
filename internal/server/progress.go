package server

import (
	"context"
	"io"
	"time"
)

// progressReader wraps an io.Reader, emitting throttled byte-count callbacks
// and aborting when its context is cancelled (so /cancel stops a live write).
type progressReader struct {
	r        io.Reader
	total    int64
	read     int64
	ctx      context.Context
	emit     func(received int64)
	lastEmit time.Time
}

func (p *progressReader) Read(b []byte) (int, error) {
	select {
	case <-p.ctx.Done():
		return 0, p.ctx.Err()
	default:
	}
	n, err := p.r.Read(b)
	p.read += int64(n)
	// Throttle to ~10 emits/sec, but always emit on EOF so the bar reaches 100%.
	if err == io.EOF || time.Since(p.lastEmit) > 100*time.Millisecond {
		p.lastEmit = time.Now()
		if p.emit != nil {
			p.emit(p.read)
		}
	}
	return n, err
}
