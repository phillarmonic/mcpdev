package port

import "time"

// Clock is an injectable time source for timeouts and watch debounce.
type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
}
