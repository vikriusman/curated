package config

import (
	"strings"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://notes:s3cret@db:5432/notes")
	c, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Port != 8080 || c.FaultRate != 0 || c.HealthInterval.String() != "5s" {
		t.Fatalf("unexpected defaults: %+v", c)
	}
}

func TestLoadFailsFastAndReportsEveryProblem(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("PORT", "abc")
	t.Setenv("FAULT_RATE", "2")
	_, err := Load()
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"DATABASE_URL is required", "PORT", "FAULT_RATE"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

func TestTimeoutMustFitInsideInterval(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://notes@db/notes")
	t.Setenv("HEALTH_INTERVAL", "1s")
	t.Setenv("DEPENDENCY_TIMEOUT", "2s")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "DEPENDENCY_TIMEOUT") {
		t.Fatalf("expected timeout/interval error, got %v", err)
	}
}

func TestRedactedHidesPassword(t *testing.T) {
	c := Config{DatabaseURL: "postgres://notes:s3cret@db:5432/notes"}
	if got := c.Redacted()["database_url"].(string); strings.Contains(got, "s3cret") {
		t.Fatalf("password leaked: %s", got)
	}
}
