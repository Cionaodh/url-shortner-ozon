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

func Migrations(log *slog.Logger) {
	log.Info("Configuring migrations...")
	pgUrl, ok := os.LookupEnv("PG_CONN")
	if !ok || len(pgUrl) == 0 {
		log.Error("Migrations error - PG_CONN not specified")
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
		log.Info(fmt.Sprintf("Postgres trying to connect, attempts left: %d", connAttempts))
		connAttempts--
	}

	if err != nil {
		log.Error(fmt.Errorf("app - init - migrate.New: %w", err).Error())
		os.Exit(1)
	}
	defer mgrt.Close()

	if err = mgrt.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Error(fmt.Errorf("app - init - mgrt.Up: %w", err).Error())
		os.Exit(1)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		log.Info("Migration no change...")
		return
	}

	log.Info("Migration successful up...")
}
