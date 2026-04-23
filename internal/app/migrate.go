package app

import (
	"errors"
	"fmt"
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

var (
	ErrConnectionStr  = errors.New("postgres connection string is not set")
	ErrInitMigrations = errors.New("failed to initialize migrations")
	ErrUpMigrations   = errors.New("failed to apply migrations")
)

func Migrations(log *slog.Logger) error {
	log = log.With("component", "migrations")

	log.Info("starting database migrations")

	pgUrl, ok := os.LookupEnv("PG_CONN")
	if !ok || len(pgUrl) == 0 {
		return ErrConnectionStr
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

		log.Info("failed to connect to postgres, retrying", slog.Int("attempts_left", connAttempts-1), slog.Any("error", err))
		time.Sleep(defaultTimeout)

		connAttempts--
	}

	if err != nil {
		return fmt.Errorf("%w: %w", ErrInitMigrations, err)
	}
	defer mgrt.Close()

	if err = mgrt.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("%w: %w", ErrUpMigrations, err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		log.Info("no new migrations to apply")
		return nil
	}

	log.Info("migration successful up")
	return nil
}
