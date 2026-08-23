// Package memory provides thread-safe, in-memory implementations of every
// outbound port. They back the unit/acceptance tests and let the service
// run locally with no database at all.
package memory

import "time"

// SystemClock implements ports.Clock using wall-clock time.
type SystemClock struct{}

// Now returns the current wall-clock time.
func (SystemClock) Now() time.Time { return time.Now() }

// FixedClock implements ports.Clock with a fixed, settable time, for tests.
type FixedClock struct {
	t time.Time
}

// NewFixedClock builds a FixedClock pinned to t.
func NewFixedClock(t time.Time) *FixedClock {
	return &FixedClock{t: t}
}

// Now returns the pinned time.
func (c *FixedClock) Now() time.Time { return c.t }

// Advance moves the pinned time forward by d.
func (c *FixedClock) Advance(d time.Duration) {
	c.t = c.t.Add(d)
}
