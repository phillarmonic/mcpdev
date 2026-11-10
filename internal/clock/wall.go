// Package clock implements port.Clock with the wall clock.
package clock

import (
	"time"

	"github.com/phillarmonic/mcpdev/internal/port"
)

// Wall is the production clock.
type Wall struct{}

var _ port.Clock = Wall{}

func (Wall) Now() time.Time {
	return time.Now()
}

func (Wall) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}
