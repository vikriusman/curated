// Package config reads every setting from the environment (12-factor III).
//
// Rules this package enforces:
//   - No config files, no per-environment code paths: the same binary runs
//     locally, in CI and in every cluster; only the environment differs.
//   - Required settings fail fast at startup with a clear message, instead of
//     failing later on the first request.
//   - Secrets are never logged: Redacted() is what goes to the startup log.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port              int           // PORT: public HTTP (12-factor VII)
	DatabaseURL       string        // DATABASE_URL: backing service as an attached resource (IV)
	LogLevel          string        // LOG_LEVEL: debug | info | warn | error
	HealthInterval    time.Duration // HEALTH_INTERVAL: how often dependencies are checked
	DependencyTimeout time.Duration // DEPENDENCY_TIMEOUT: max wait per dependency check
	DrainDelay        time.Duration // DRAIN_DELAY: unready-before-shutdown window (IX)
	ShutdownTimeout   time.Duration // SHUTDOWN_TIMEOUT: max time to finish in-flight requests (IX)
	FaultRate         float64       // FAULT_RATE: chaos knob, share of API requests answered with 500
}

func Load() (Config, error) {
	var errs []error
	c := Config{
		Port:              intEnv("PORT", 8080, &errs),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		LogLevel:          strEnv("LOG_LEVEL", "info"),
		HealthInterval:    durEnv("HEALTH_INTERVAL", 5*time.Second, &errs),
		DependencyTimeout: durEnv("DEPENDENCY_TIMEOUT", time.Second, &errs),
		DrainDelay:        durEnv("DRAIN_DELAY", 5*time.Second, &errs),
		ShutdownTimeout:   durEnv("SHUTDOWN_TIMEOUT", 15*time.Second, &errs),
		FaultRate:         floatEnv("FAULT_RATE", 0, &errs),
	}

	if c.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if c.FaultRate < 0 || c.FaultRate > 1 {
		errs = append(errs, fmt.Errorf("FAULT_RATE must be between 0 and 1, got %v", c.FaultRate))
	}
	if c.DependencyTimeout >= c.HealthInterval {
		errs = append(errs, errors.New("DEPENDENCY_TIMEOUT must be shorter than HEALTH_INTERVAL"))
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		errs = append(errs, fmt.Errorf("LOG_LEVEL must be debug|info|warn|error, got %q", c.LogLevel))
	}
	return c, errors.Join(errs...)
}

// Redacted returns the effective config with credentials masked, for logging.
func (c Config) Redacted() map[string]any {
	return map[string]any{
		"port":               c.Port,
		"database_url":       redactURL(c.DatabaseURL),
		"log_level":          c.LogLevel,
		"health_interval":    c.HealthInterval.String(),
		"dependency_timeout": c.DependencyTimeout.String(),
		"drain_delay":        c.DrainDelay.String(),
		"shutdown_timeout":   c.ShutdownTimeout.String(),
		"fault_rate":         c.FaultRate,
	}
}

func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}
	if _, ok := u.User.Password(); ok {
		u.User = url.UserPassword(u.User.Username(), "REDACTED")
	}
	return u.String()
}

func strEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func intEnv(key string, def int, errs *[]error) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s: %q is not an integer", key, v))
		return def
	}
	return n
}

func floatEnv(key string, def float64, errs *[]error) float64 {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s: %q is not a number", key, v))
		return def
	}
	return f
}

func durEnv(key string, def time.Duration, errs *[]error) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s: %q is not a duration (e.g. 5s)", key, v))
		return def
	}
	return d
}
