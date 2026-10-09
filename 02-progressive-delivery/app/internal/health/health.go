// Package health answers one question for the platform: can this process
// serve requests right now? Ready means every required dependency answers.
//
// Checks run in the background on a fixed interval and the result is cached,
// so probes from kubelet (every few seconds, from every replica) never
// multiply into load on the database.
package health

import (
	"context"
	"sync/atomic"
	"time"
)

// Check is one required dependency.
type Check struct {
	Name  string
	Probe func(ctx context.Context) error
}

type Result struct {
	Name      string  `json:"name"`
	OK        bool    `json:"ok"`
	LatencyMS float64 `json:"latency_ms"`
	Error     string  `json:"error,omitempty"`
}

type Report struct {
	Ready     bool      `json:"ready"`
	CheckedAt time.Time `json:"checked_at"`
	Checks    []Result  `json:"checks"`
}

type Checker struct {
	checks   []Check
	interval time.Duration
	timeout  time.Duration
	latest   atomic.Pointer[Report]
}

func New(interval, timeout time.Duration, checks ...Check) *Checker {
	c := &Checker{checks: checks, interval: interval, timeout: timeout}
	// Not ready until the first round completes.
	c.latest.Store(&Report{Ready: false, CheckedAt: time.Now()})
	return c
}

// Run checks once immediately, then every interval until ctx is done.
func (c *Checker) Run(ctx context.Context) {
	c.RunOnce(ctx)
	t := time.NewTicker(c.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.RunOnce(ctx)
		}
	}
}

func (c *Checker) RunOnce(ctx context.Context) {
	results := make([]Result, len(c.checks))
	done := make(chan struct{}, len(c.checks))
	for i, chk := range c.checks {
		go func() {
			defer func() { done <- struct{}{} }()
			cctx, cancel := context.WithTimeout(ctx, c.timeout)
			defer cancel()
			start := time.Now()
			err := chk.Probe(cctx)
			r := Result{Name: chk.Name, OK: err == nil, LatencyMS: float64(time.Since(start).Microseconds()) / 1000}
			if err != nil {
				r.Error = err.Error()
			}
			results[i] = r
		}()
	}
	for range c.checks {
		<-done
	}
	ready := true
	for _, r := range results {
		ready = ready && r.OK
	}
	c.latest.Store(&Report{Ready: ready, CheckedAt: time.Now(), Checks: results})
}

func (c *Checker) Latest() Report { return *c.latest.Load() }
