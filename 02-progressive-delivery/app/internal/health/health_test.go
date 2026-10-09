package health

import (
	"context"
	"errors"
	"testing"
	"time"
)

func ok(context.Context) error   { return nil }
func fail(context.Context) error { return errors.New("down") }

func TestNotReadyUntilFirstCheck(t *testing.T) {
	c := New(time.Hour, time.Second, Check{Name: "db", Probe: ok})
	if c.Latest().Ready {
		t.Fatal("must not report ready before the first check ran")
	}
	c.RunOnce(context.Background())
	if !c.Latest().Ready {
		t.Fatal("must be ready after a passing check")
	}
}

func TestAnyFailingCheckMeansNotReady(t *testing.T) {
	c := New(time.Hour, time.Second, Check{Name: "db", Probe: ok}, Check{Name: "cache", Probe: fail})
	c.RunOnce(context.Background())
	r := c.Latest()
	if r.Ready {
		t.Fatal("one failing dependency must make the service not ready")
	}
	if r.Checks[1].Error == "" {
		t.Fatal("the failing check must carry its error for operators")
	}
}

func TestSlowDependencyTimesOut(t *testing.T) {
	slow := func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }
	c := New(time.Hour, 50*time.Millisecond, Check{Name: "db", Probe: slow})
	start := time.Now()
	c.RunOnce(context.Background())
	if time.Since(start) > time.Second {
		t.Fatal("a hanging dependency must not hang the health check")
	}
	if c.Latest().Ready {
		t.Fatal("a timed-out dependency must make the service not ready")
	}
}
