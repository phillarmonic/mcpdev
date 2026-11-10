package clock

import (
	"sync"
	"time"

	"github.com/phillarmonic/mcpdev/internal/port"
)

// Virtual is an injectable clock that advances only when Advance is called.
type Virtual struct {
	now     time.Time
	waiters []virtualWaiter
	mu      sync.Mutex
}

type virtualWaiter struct {
	at time.Time
	ch chan time.Time
}

var _ port.Clock = (*Virtual)(nil)

// NewVirtual returns a clock starting at now. A zero now is Unix epoch UTC.
func NewVirtual(now time.Time) *Virtual {
	if now.IsZero() {
		now = time.Unix(0, 0).UTC()
	}
	return &Virtual{now: now}
}

func (v *Virtual) Now() time.Time {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.now
}

func (v *Virtual) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	v.mu.Lock()
	defer v.mu.Unlock()
	if d <= 0 {
		ch <- v.now
		return ch
	}
	v.waiters = append(v.waiters, virtualWaiter{at: v.now.Add(d), ch: ch})
	return ch
}

// Advance moves the clock forward and releases waiters whose deadlines have been reached.
func (v *Virtual) Advance(d time.Duration) {
	if d < 0 {
		d = 0
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.now = v.now.Add(d)
	var rest []virtualWaiter
	for _, w := range v.waiters {
		if w.at.After(v.now) {
			rest = append(rest, w)
			continue
		}
		w.ch <- v.now
	}
	v.waiters = rest
}
