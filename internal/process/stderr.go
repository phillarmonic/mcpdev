package process

import "sync"

// boundWriter keeps a bounded prefix of child stderr and counts dropped bytes.
type boundWriter struct {
	buf     []byte
	max     int
	dropped int
	mu      sync.Mutex
}

func newBoundWriter(limit int) *boundWriter {
	if limit <= 0 {
		limit = defaultStderrLimit
	}
	return &boundWriter{max: limit}
}

func (b *boundWriter) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	room := b.max - len(b.buf)
	if room > 0 {
		if room > len(p) {
			room = len(p)
		}
		b.buf = append(b.buf, p[:room]...)
	} else {
		room = 0
	}
	if len(p) > room {
		b.dropped += len(p) - room
	}
	return len(p), nil
}

func (b *boundWriter) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}

func (b *boundWriter) Dropped() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.dropped
}
