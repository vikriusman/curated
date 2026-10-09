// Command notes is one binary with two process types (12-factor VIII, XII):
//
//	notes serve    long-running web process, scaled by replicas
//	notes migrate  one-off admin process, run as a Job before a rollout
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/vikriusman/curated/02-progressive-delivery/app/internal/config"
	"github.com/vikriusman/curated/02-progressive-delivery/app/internal/health"
	"github.com/vikriusman/curated/02-progressive-delivery/app/internal/migrate"
	"github.com/vikriusman/curated/02-progressive-delivery/app/internal/server"
	"github.com/vikriusman/curated/02-progressive-delivery/app/internal/store"
)

// version is stamped at build time (-ldflags "-X main.version=..."). It belongs
// to the build, not to the config: the same image always reports the same version.
var version = "dev"

func main() {
	slog.SetDefault(newLogger("info").With("version", version))
	if err := run(); err != nil {
		// Logs go to stdout/stderr as an event stream; routing them is the platform's job (XI).
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	if cmd == "version" {
		fmt.Println(version)
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("invalid configuration:\n%w", err)
	}
	log := newLogger(cfg.LogLevel).With("version", version)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	switch cmd {
	case "serve":
		return serve(ctx, cfg, log)
	case "migrate":
		return migrate.Run(ctx, cfg.DatabaseURL, log)
	default:
		return fmt.Errorf("unknown command %q (want serve | migrate | version)", cmd)
	}
}

func serve(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	log.Info("starting", "config", cfg.Redacted())

	st, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer st.Close()

	// Dependencies the service cannot work without, declared in one place.
	checker := health.New(cfg.HealthInterval, cfg.DependencyTimeout,
		health.Check{Name: "postgres", Probe: st.Ping},
	)
	go checker.Run(ctx)

	srv := server.New(st, checker, log, version, cfg.FaultRate)
	public := &http.Server{
		Addr:              net.JoinHostPort("", strconv.Itoa(cfg.Port)),
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errc := make(chan error, 1)
	go func() {
		if err := public.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
		}
	}()
	log.Info("listening", "port", cfg.Port)

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	// Disposability (IX): stop taking new traffic, let Kubernetes and Envoy
	// notice, finish in-flight requests, then exit. No preStop hook or shell
	// needed, so the image can stay distroless.
	log.Info("SIGTERM received, draining", "drain_delay", cfg.DrainDelay.String())
	srv.StartDraining()
	time.Sleep(cfg.DrainDelay)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	err = public.Shutdown(shutdownCtx)
	log.Info("stopped", "err", err)
	return err
}

func newLogger(level string) *slog.Logger {
	var l slog.Level
	_ = l.UnmarshalText([]byte(level))
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l}))
}
