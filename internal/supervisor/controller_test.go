package supervisor

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/port"
)

func TestSwapPromotesOneGeneration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLab()
		c := New(Options{Starter: &fakeStarter{lab: l, stderr: "boot"}, Probe: probeOK(l)})
		t.Cleanup(func() { _ = c.Close(context.Background()) })
		gen, err := c.Begin()
		if err != nil {
			t.Fatal(err)
		}
		out, err := c.Swap(context.Background(), gen, SwapSpec{BuildLog: "ok compile"})
		if err != nil {
			t.Fatal(err)
		}
		if out.ID != 1 || out.State != StateReady || !out.HasProtocol || out.BuildLog != "ok compile" {
			t.Fatalf("%+v", out)
		}
		if out.Protocol.ProtocolVersion != "2025-11-25" {
			t.Fatalf("protocol %+v", out.Protocol)
		}
		v := c.Snapshot()
		if v.State != StateReady || v.Current == nil || v.Current.ID != 1 || v.Incoming != nil {
			t.Fatalf("view %+v", v)
		}
		if l.liveCount() != 1 {
			t.Fatalf("live %d", l.liveCount())
		}
	})
}

func TestSecondSwapReplacesPrevious(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLab()
		c := New(Options{Starter: &fakeStarter{lab: l}, Probe: probeOK(l)})
		t.Cleanup(func() { _ = c.Close(context.Background()) })
		first := mustSwap(t, c, SwapSpec{})
		second := mustSwap(t, c, SwapSpec{})
		if first.ID == second.ID {
			t.Fatal("same generation")
		}
		v := c.Snapshot()
		if v.Current.ID != second.ID || l.liveCount() != 1 {
			t.Fatalf("current %d live %d", v.Current.ID, l.liveCount())
		}
	})
}

func TestStaleSwapRejected(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLab()
		c := New(Options{Starter: &fakeStarter{lab: l}, Probe: probeOK(l)})
		t.Cleanup(func() { _ = c.Close(context.Background()) })
		old, err := c.Begin()
		if err != nil {
			t.Fatal(err)
		}
		if _, beginErr := c.Begin(); beginErr != nil {
			t.Fatal(beginErr)
		}
		_, err = c.Swap(context.Background(), old, SwapSpec{})
		var se StaleError
		if !errors.As(err, &se) || se.Generation != old {
			t.Fatalf("err %v", err)
		}
		if errclass.ExitCode(err) != errclass.ExitIncomplete {
			t.Fatalf("class %d", errclass.ExitCode(err))
		}
	})
}

func TestRapidRebuildCancelsProbe(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLab()
		release := make(chan struct{})
		started := make(chan struct{}, 2)
		probe := func(ctx context.Context, proc port.Proc) (port.Session, error) {
			started <- struct{}{}
			select {
			case <-release:
				return probeOK(l)(ctx, proc)
			case <-ctx.Done():
				return nil, errclass.Connection{Err: ctx.Err()}
			}
		}
		c := New(Options{Starter: &fakeStarter{lab: l}, Probe: probe})
		t.Cleanup(func() { _ = c.Close(context.Background()) })
		g1, err := c.Begin()
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() {
			_, swapErr := c.Swap(context.Background(), g1, SwapSpec{})
			done <- swapErr
		}()
		synctest.Wait()
		select {
		case <-started:
		default:
			t.Fatal("probe not started")
		}
		g2, err := c.Begin()
		if err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		if err := <-done; err == nil {
			t.Fatal("expected first swap to fail")
		}
		close(release)
		if _, err := c.Swap(context.Background(), g2, SwapSpec{}); err != nil {
			t.Fatal(err)
		}
		if l.liveCount() != 1 {
			t.Fatalf("live %d", l.liveCount())
		}
		v := c.Snapshot()
		if v.Current == nil || v.Current.ID != g2 {
			t.Fatalf("current %+v", v.Current)
		}
	})
}

func TestFailedProbeKeepsLastGood(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLab()
		var fail atomic.Bool
		probe := func(ctx context.Context, proc port.Proc) (port.Session, error) {
			if fail.Load() {
				return nil, errclass.Connection{Err: errors.New("handshake hang")}
			}
			return probeOK(l)(ctx, proc)
		}
		c := New(Options{Starter: &fakeStarter{lab: l, stderr: "child-err"}, Probe: probe})
		t.Cleanup(func() { _ = c.Close(context.Background()) })
		good := mustSwap(t, c, SwapSpec{BuildLog: "good"})
		fail.Store(true)
		g, err := c.Begin()
		if err != nil {
			t.Fatal(err)
		}
		_, err = c.Swap(context.Background(), g, SwapSpec{BuildLog: "bad"})
		if err == nil {
			t.Fatal("expected probe failure")
		}
		v := c.Snapshot()
		if v.State != StateReady || v.Current.ID != good.ID || l.liveCount() != 1 {
			t.Fatalf("view %+v live %d", v, l.liveCount())
		}
		if v.LastFailed == nil || v.LastFailed.Stderr != "child-err" || v.LastFailed.TerminateReason == "" {
			t.Fatalf("failed %+v", v.LastFailed)
		}
	})
}

func TestReplaceInPlaceStopsBeforeStart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLab()
		var liveAtStart atomic.Int32
		liveAtStart.Store(-1)
		c := New(Options{
			Starter:        &fakeStarter{lab: l, liveAtStart: &liveAtStart},
			Probe:          probeOK(l),
			ReplaceInPlace: true,
		})
		t.Cleanup(func() { _ = c.Close(context.Background()) })
		mustSwap(t, c, SwapSpec{})
		mustSwap(t, c, SwapSpec{})
		if liveAtStart.Load() != 0 {
			t.Fatalf("live at start %d", liveAtStart.Load())
		}
		if l.liveCount() != 1 {
			t.Fatalf("live %d", l.liveCount())
		}
	})
}

func TestFailBuildLeavesLastGood(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLab()
		c := New(Options{Starter: &fakeStarter{lab: l}, Probe: probeOK(l)})
		t.Cleanup(func() { _ = c.Close(context.Background()) })
		good := mustSwap(t, c, SwapSpec{})
		g, err := c.Begin()
		if err != nil {
			t.Fatal(err)
		}
		if err := c.FailBuild(g, "compiler boom", "nonzero"); err != nil {
			t.Fatal(err)
		}
		v := c.Snapshot()
		if v.State != StateReady || v.Current.ID != good.ID {
			t.Fatalf("%+v", v)
		}
		if v.LastFailed == nil || v.LastFailed.BuildLog != "compiler boom" {
			t.Fatalf("failed %+v", v.LastFailed)
		}
		if l.liveCount() != 1 {
			t.Fatalf("live %d", l.liveCount())
		}
	})
}

func TestStartupHangCancel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLab()
		hang := make(chan struct{})
		c := New(Options{Starter: &fakeStarter{lab: l, hangStart: hang}, Probe: probeOK(l)})
		t.Cleanup(func() { _ = c.Close(context.Background()) })
		g, err := c.Begin()
		if err != nil {
			t.Fatal(err)
		}
		swapCtx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { _, err := c.Swap(swapCtx, g, SwapSpec{}); done <- err }()
		synctest.Wait()
		cancel()
		synctest.Wait()
		if err := <-done; err == nil {
			t.Fatal("expected cancel")
		}
		if l.liveCount() != 0 {
			t.Fatalf("live %d", l.liveCount())
		}
	})
}

func TestCloseDuringProbe(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLab()
		probe := func(ctx context.Context, _ port.Proc) (port.Session, error) {
			<-ctx.Done()
			return nil, errclass.Connection{Err: ctx.Err()}
		}
		c := New(Options{Starter: &fakeStarter{lab: l}, Probe: probe})
		g, err := c.Begin()
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() { _, err := c.Swap(context.Background(), g, SwapSpec{}); done <- err }()
		synctest.Wait()
		if err := c.Close(context.Background()); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		<-done //nolint:errcheck // Swap races Close; the test asserts live state, not its error.
		if l.liveCount() != 0 {
			t.Fatalf("live %d", l.liveCount())
		}
		if c.Snapshot().State != StateClosed {
			t.Fatalf("state %s", c.Snapshot().State)
		}
	})
}

func TestCrashDuringProbe(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLab()
		probe := func(ctx context.Context, proc port.Proc) (port.Session, error) {
			_ = proc.Close(ctx)
			return nil, errclass.Connection{Err: errors.New("child exited")}
		}
		c := New(Options{Starter: &fakeStarter{lab: l, stderr: "panic"}, Probe: probe})
		t.Cleanup(func() { _ = c.Close(context.Background()) })
		g, err := c.Begin()
		if err != nil {
			t.Fatal(err)
		}
		_, err = c.Swap(context.Background(), g, SwapSpec{})
		if err == nil {
			t.Fatal("expected crash")
		}
		v := c.Snapshot()
		if v.State != StateFailed || v.LastFailed == nil || v.LastFailed.Stderr != "panic" {
			t.Fatalf("%+v", v)
		}
		if l.liveCount() != 0 {
			t.Fatalf("live %d", l.liveCount())
		}
	})
}

func TestCloseStopsReadyGeneration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLab()
		c := New(Options{Starter: &fakeStarter{lab: l}, Probe: probeOK(l)})
		mustSwap(t, c, SwapSpec{})
		if err := c.Close(context.Background()); err != nil {
			t.Fatal(err)
		}
		if l.liveCount() != 0 {
			t.Fatalf("live %d", l.liveCount())
		}
		if _, err := c.Begin(); errclass.ExitCode(err) != errclass.ExitConnection {
			t.Fatalf("begin after close %v", err)
		}
	})
}

func TestGenerationsIncrease(t *testing.T) {
	c := New(Options{})
	a, _ := c.Begin()
	b, _ := c.Begin()
	if a >= b {
		t.Fatalf("%d then %d", a, b)
	}
	_ = c.Close(context.Background())
}

func mustSwap(t *testing.T, c *Controller, spec SwapSpec) *Generation {
	t.Helper()
	g, err := c.Begin()
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.Swap(context.Background(), g, spec)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestProbeTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLab()
		probe := func(ctx context.Context, _ port.Proc) (port.Session, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}
		c := New(Options{Starter: &fakeStarter{lab: l}, Probe: probe, ProbeTimeout: 50 * time.Millisecond})
		t.Cleanup(func() { _ = c.Close(context.Background()) })
		g, err := c.Begin()
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() { _, err := c.Swap(context.Background(), g, SwapSpec{}); done <- err }()
		time.Sleep(50 * time.Millisecond)
		synctest.Wait()
		if err := <-done; err == nil {
			t.Fatal("expected timeout")
		}
		if l.liveCount() != 0 {
			t.Fatalf("live %d", l.liveCount())
		}
	})
}
