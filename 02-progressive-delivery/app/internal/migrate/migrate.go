// Package migrate applies schema changes as a one-off admin process (12-factor XII):
// same binary, same image, same config as the web process, run as `notes migrate`
// in a Kubernetes Job before a rollout. The web process never migrates on boot,
// so scaling to N replicas never means N concurrent migrations.
package migrate

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed sql/*.sql
var files embed.FS

// Arbitrary constant: serialises concurrent `notes migrate` runs cluster-wide.
const lockID = 74_201_002

type migration struct {
	version int
	name    string
	sql     string
}

func Run(ctx context.Context, databaseURL string, log *slog.Logger) error {
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close(context.Background())

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", lockID); err != nil {
		return fmt.Errorf("lock: %w", err)
	}
	defer conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", lockID) //nolint:errcheck

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    INT PRIMARY KEY,
		name       TEXT NOT NULL,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return fmt.Errorf("bookkeeping table: %w", err)
	}

	all, err := load()
	if err != nil {
		return err
	}

	applied := 0
	for _, m := range all {
		var exists bool
		if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)", m.version).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}
		// One transaction per migration: a failure leaves the schema at the last good version.
		err := pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, m.sql); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version, name) VALUES ($1, $2)", m.version, m.name)
			return err
		})
		if err != nil {
			return fmt.Errorf("migration %04d %s: %w", m.version, m.name, err)
		}
		log.Info("migration applied", "migration", m.version, "name", m.name)
		applied++
	}
	log.Info("schema up to date", "applied", applied, "known", len(all))
	return nil
}

func load() ([]migration, error) {
	entries, err := fs.ReadDir(files, "sql")
	if err != nil {
		return nil, err
	}
	var out []migration
	for _, e := range entries {
		name := strings.TrimSuffix(e.Name(), ".sql")
		num, _, ok := strings.Cut(name, "_")
		v, err := strconv.Atoi(num)
		if !ok || err != nil {
			return nil, fmt.Errorf("bad migration file name %q (want NNNN_name.sql)", e.Name())
		}
		body, err := files.ReadFile("sql/" + e.Name())
		if err != nil {
			return nil, err
		}
		out = append(out, migration{version: v, name: name, sql: string(body)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}
