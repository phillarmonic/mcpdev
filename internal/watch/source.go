package watch

import "context"

// Source is an injectable stream of normalized file events.
type Source interface {
	Events() <-chan Event
	Errors() <-chan error
}

// Fake is a test event source. Sends on Events block until Loop receives them.
type Fake struct {
	events chan Event
	errors chan error
}

// NewFake returns an unbuffered event source.
func NewFake() *Fake {
	return &Fake{
		events: make(chan Event),
		errors: make(chan error, 1),
	}
}

func (f *Fake) Events() <-chan Event { return f.events }
func (f *Fake) Errors() <-chan error { return f.errors }

// Send delivers one event. It blocks until Loop reads it or ctx is done.
func (f *Fake) Send(ctx context.Context, ev Event) error {
	select {
	case f.events <- ev:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Fail delivers a watcher error.
func (f *Fake) Fail(ctx context.Context, err error) error {
	select {
	case f.errors <- err:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
