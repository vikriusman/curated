package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vikriusman/curated/02-progressive-delivery/app/internal/health"
	"github.com/vikriusman/curated/02-progressive-delivery/app/internal/store"
)

type fakeNotes struct{ items []store.Note }

func (f *fakeNotes) Create(_ context.Context, body string) (store.Note, error) {
	n := store.Note{ID: int64(len(f.items) + 1), Body: body, CreatedAt: time.Now()}
	f.items = append(f.items, n)
	return n, nil
}
func (f *fakeNotes) List(context.Context, int) ([]store.Note, error) { return f.items, nil }

type fixedHealth bool

func (f fixedHealth) Latest() health.Report { return health.Report{Ready: bool(f)} }

func newTest(ready bool, fault float64) *Server {
	return New(&fakeNotes{}, fixedHealth(ready), slog.New(slog.NewTextHandler(io.Discard, nil)), "test", fault)
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestReadiness(t *testing.T) {
	if c := do(newTest(true, 0).Handler(), "GET", "/readyz", "").Code; c != 200 {
		t.Errorf("ready: got %d, want 200", c)
	}
	if c := do(newTest(false, 0).Handler(), "GET", "/readyz", "").Code; c != 503 {
		t.Errorf("not ready: got %d, want 503", c)
	}
}

func TestDrainingMakesNotReadyButStaysAlive(t *testing.T) {
	s := newTest(true, 0)
	s.StartDraining()
	h := s.Handler()
	if c := do(h, "GET", "/readyz", "").Code; c != 503 {
		t.Fatalf("draining: got %d, want 503", c)
	}
	if c := do(h, "GET", "/healthz", "").Code; c != 200 {
		t.Fatalf("liveness must stay 200 while draining, got %d", c)
	}
}

func TestLivenessIgnoresDependencies(t *testing.T) {
	if c := do(newTest(false, 0).Handler(), "GET", "/healthz", "").Code; c != 200 {
		t.Fatalf("liveness must stay 200 when dependencies fail, got %d", c)
	}
}

// A faulty release passes every probe; only traffic analysis can catch it.
func TestFaultRateBreaksAPIButNotProbes(t *testing.T) {
	h := newTest(true, 1).Handler()
	if c := do(h, "GET", "/api/notes", "").Code; c != 500 {
		t.Fatalf("api with FAULT_RATE=1: got %d, want 500", c)
	}
	for _, p := range []string{"/healthz", "/readyz", "/version"} {
		if c := do(h, "GET", p, "").Code; c != 200 {
			t.Fatalf("%s must not be affected by faults, got %d", p, c)
		}
	}
}

func TestCreateAndList(t *testing.T) {
	h := newTest(true, 0).Handler()
	if c := do(h, "POST", "/api/notes", `{"body":"hello"}`).Code; c != 201 {
		t.Fatalf("create: got %d", c)
	}
	if c := do(h, "POST", "/api/notes", `{"body":""}`).Code; c != 422 {
		t.Fatalf("empty body: got %d, want 422", c)
	}
	if !strings.Contains(do(h, "GET", "/api/notes", "").Body.String(), "hello") {
		t.Fatal("list missing note")
	}
}
