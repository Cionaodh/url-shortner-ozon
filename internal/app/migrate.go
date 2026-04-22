package app

import (
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	defaultAttempts = 15
	defaultTimeout  = time.Second
)

func Migrations(log *slog.Logger) {
	log = log.With("component", "mirgate")

	log.Info("starting database migrations")

	pgUrl, ok := os.LookupEnv("PG_CONN")
	if !ok || len(pgUrl) == 0 {
		log.Error("postgres connection string is not set", slog.String("env", "PG_CONN"))
		os.Exit(1)
	}
	pgUrl += "?sslmode=disable"

	var (
		connAttempts = defaultAttempts
		err          error
		mgrt         *migrate.Migrate
	)

	for connAttempts > 0 {
		mgrt, err = migrate.New("file://migrations", pgUrl)
		if err == nil {
			break
		}

		time.Sleep(defaultTimeout)
		log.Warn("failed to connect to postgres, retrying", slog.Int("attempts_left", connAttempts-1), slog.Any("error", err))
		connAttempts--
	}

	if err != nil {
		log.Error("failed to initialize migrations", slog.Any("error", err))
		os.Exit(1)
	}
	defer mgrt.Close()

	if err = mgrt.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Error("failed to apply migrations", slog.Any("error", err))
		os.Exit(1)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		log.Info("no new migrations to apply")
		return
	}

	log.Info("migration successful up")
}
