package watch

import (
	"context"
	"slices"
	"time"

	"github.com/phillarmonic/mcpdev/internal/clock"
	"github.com/phillarmonic/mcpdev/internal/port"
)

const defaultDebounce = 200 * time.Millisecond

// BuildFunc runs one coalesced rebuild. A non-nil error does not stop Loop.
type BuildFunc func(ctx context.Context, batch []Event) error

// Report is one finished rebuild attempt.
type Report struct {
	Err   error
	Batch []Event
}

// Loop coalesces file events onto an injectable clock and serializes builds.
type Loop struct {
	Source   Source
	Clock    port.Clock
	Build    BuildFunc
	OnReport func(Report)
	Matcher  Matcher
	Debounce time.Duration
}

// Run watches until ctx is cancelled or the source reports a fatal error.
func (l Loop) Run(ctx context.Context) error {
	if l.Source == nil {
		<-ctx.Done()
		return ctx.Err()
	}
	if len(l.Matcher.includes) == 0 && len(l.Matcher.excludes) == 0 {
		l.Matcher = NewMatcher(nil, nil)
	}
	clk := l.Clock
	if clk == nil {
		clk = clock.Wall{}
	}
	debounce := l.Debounce
	if debounce <= 0 {
		debounce = defaultDebounce
	}
	var (
		gen       uint64
		armed     uint64
		timer     <-chan time.Time
		batch     []Event
		building  bool
		pending   bool
		buildDone = make(chan Report, 1)
		stopBuild context.CancelFunc
	)
	defer func() {
		if stopBuild != nil {
			stopBuild()
		}
	}()
	start := func() {
		if l.Build == nil || len(batch) == 0 || building {
			return
		}
		copied := slices.Clone(batch)
		batch = nil
		pending = false
		building = true
		bctx, cancel := context.WithCancel(ctx)
		stopBuild = cancel
		go func() {
			// rebuild: owned by Loop; joined via buildDone or parent cancel.
			defer cancel()
			err := l.Build(bctx, copied)
			select {
			case buildDone <- Report{Batch: copied, Err: err}:
			case <-ctx.Done():
			}
		}()
	}
	arm := func() {
		gen++
		armed = gen
		timer = clk.After(debounce)
	}
	for {
		select {
		case <-ctx.Done():
			if stopBuild != nil {
				stopBuild()
			}
			return ctx.Err()
		case err, ok := <-l.Source.Errors():
			if !ok {
				return ctx.Err()
			}
			if err != nil {
				return err
			}
		case ev, ok := <-l.Source.Events():
			if !ok {
				return nil
			}
			if !l.Matcher.Accept(ev.Path) {
				continue
			}
			batch = append(batch, ev)
			if building {
				pending = true
				continue
			}
			arm()
		case <-timer:
			timer = nil
			if armed != gen {
				continue
			}
			if building {
				pending = true
				continue
			}
			start()
		case report := <-buildDone:
			building = false
			if stopBuild != nil {
				stopBuild()
				stopBuild = nil
			}
			if l.OnReport != nil {
				l.OnReport(report)
			}
			if pending || len(batch) > 0 {
				start()
			}
		}
	}
}
