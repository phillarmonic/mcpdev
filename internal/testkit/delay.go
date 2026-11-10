package testkit

import "time"

// timeDelay is long enough for cancellation tests and short enough for synctest.
const timeDelay = 50 * time.Millisecond
