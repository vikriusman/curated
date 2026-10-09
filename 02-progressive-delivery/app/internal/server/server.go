// Package server is the web process: a self-contained HTTP server that binds
// to a port from the environment (12-factor VII), tells the platform whether
// it can serve (see package health), and logs one JSON line per request to
// stdout (12-factor XI); those lines are what canary analysis reads from Loki.
package server

import (
	"context"
	"encoding/json"
	"html/template"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/vikriusman/curated/02-progressive-delivery/app/internal/health"
	"github.com/vikriusman/curated/02-progressive-delivery/app/internal/store"
)

type Notes interface {
	Create(ctx context.Context, body string) (store.Note, error)
	List(ctx context.Context, limit int) ([]store.Note, error)
}

type Health interface {
	Latest() health.Report
}

type Server struct {
	notes     Notes
	health    Health
	log       *slog.Logger
	version   string
	pod       string
	faultRate float64
	draining  atomic.Bool
}

func New(notes Notes, h Health, log *slog.Logger, version string, faultRate float64) *Server {
	pod, _ := os.Hostname()
	return &Server{notes: notes, health: h, log: log, version: version, pod: pod, faultRate: faultRate}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /readyz", s.readyz)
	mux.HandleFunc("GET /version", s.versionInfo)
	mux.Handle("GET /{$}", s.faulty(http.HandlerFunc(s.index)))
	mux.Handle("POST /notes", s.faulty(http.HandlerFunc(s.createForm)))
	mux.Handle("GET /api/notes", s.faulty(http.HandlerFunc(s.listJSON)))
	mux.Handle("POST /api/notes", s.faulty(http.HandlerFunc(s.createJSON)))
	return s.logRequests(mux)
}

// StartDraining makes readiness fail so Kubernetes stops routing new traffic
// here before the server shuts down (12-factor IX).
func (s *Server) StartDraining() { s.draining.Store(true) }

// Liveness is deliberately shallow: "is this process alive?". If it checked
// the database, a database outage would make Kubernetes restart every pod.
func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

// Readiness: 200 when every required dependency answers (cached report),
// 503 otherwise or while draining. The body says why, for operators.
func (s *Server) readyz(w http.ResponseWriter, _ *http.Request) {
	report := s.health.Latest()
	ready := report.Ready && !s.draining.Load()
	code := http.StatusOK
	if !ready {
		code = http.StatusServiceUnavailable
	}
	writeJSON(w, code, map[string]any{
		"ready":      ready,
		"draining":   s.draining.Load(),
		"checked_at": report.CheckedAt,
		"checks":     report.Checks,
		"version":    s.version,
		"pod":        s.pod,
	})
}

func (s *Server) versionInfo(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"version":    s.version,
		"pod":        s.pod,
		"fault_rate": s.faultRate,
	})
}

func (s *Server) listJSON(w http.ResponseWriter, r *http.Request) {
	notes, err := s.notes.List(r.Context(), 50)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, notes)
}

func (s *Server) createJSON(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	s.create(w, r, in.Body, func(n store.Note) { writeJSON(w, http.StatusCreated, n) })
}

func (s *Server) createForm(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	s.create(w, r, r.FormValue("body"), func(store.Note) { http.Redirect(w, r, "/", http.StatusSeeOther) })
}

func (s *Server) create(w http.ResponseWriter, r *http.Request, body string, ok func(store.Note)) {
	body = strings.TrimSpace(body)
	if body == "" || len(body) > 500 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "body must be 1-500 characters"})
		return
	}
	n, err := s.notes.Create(r.Context(), body)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	ok(n)
}

var page = template.Must(template.New("index").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Notes · {{.Version}}</title>
<style>
 :root{--bg:#f6f5f2;--fg:#1d1d1b;--muted:#6b6b66;--card:#fff;--line:#e3e1db;--accent:#0f766e}
 @media (prefers-color-scheme:dark){:root{--bg:#151514;--fg:#ecebe6;--muted:#9b9a94;--card:#1f1f1d;--line:#33322f;--accent:#2dd4bf}}
 body{margin:0;font:15px/1.5 system-ui,sans-serif;background:var(--bg);color:var(--fg)}
 main{max-width:720px;margin:0 auto;padding:24px 16px}
 .meta{font:12px ui-monospace,monospace;background:var(--card);border:1px solid var(--line);border-radius:6px;padding:8px 12px}
 .meta b{color:var(--accent)} form{display:flex;gap:8px;margin:16px 0}
 input{flex:1;font:inherit;padding:8px 10px;border:1px solid var(--line);border-radius:6px;background:var(--card);color:var(--fg)}
 button{font:inherit;padding:8px 14px;border:0;border-radius:6px;background:var(--accent);color:#fff}
 li{padding:8px 0;border-bottom:1px solid var(--line)} small{color:var(--muted)}
</style></head><body><main>
<h1>Notes</h1>
<div class="meta">version <b>{{.Version}}</b> · pod <b>{{.Pod}}</b></div>
<form method="post" action="/notes"><input name="body" maxlength="500" placeholder="Write a note" required><button>Add</button></form>
<ul>{{range .Notes}}<li>{{.Body}} <small>#{{.ID}} · {{.CreatedAt.Format "2006-01-02 15:04:05"}}</small></li>{{else}}<li><small>No notes yet.</small></li>{{end}}</ul>
</main></body></html>`))

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	notes, err := s.notes.List(r.Context(), 20)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = page.Execute(w, map[string]any{"Version": s.version, "Pod": s.pod, "Notes": notes})
}

// faulty is a chaos knob driven purely by config (FAULT_RATE): it simulates a
// release that is healthy by every probe yet broken for users. Only traffic
// analysis (here: error rate from the request logs in Loki) can catch it.
func (s *Server) faulty(next http.Handler) http.Handler {
	if s.faultRate <= 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rand.Float64() < s.faultRate {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "injected fault"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) fail(w http.ResponseWriter, r *http.Request, err error) {
	s.log.Error("request failed", "method", r.Method, "path", r.URL.Path, "err", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
}

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}

// logRequests writes one structured line per request to stdout (12-factor XI).
func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(rec, r)
		if r.Pattern == "GET /healthz" || r.Pattern == "GET /readyz" {
			return // probes would drown the log
		}
		s.log.Info("request", "method", r.Method, "route", r.Pattern, "code", rec.code,
			"duration_ms", time.Since(start).Milliseconds())
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
