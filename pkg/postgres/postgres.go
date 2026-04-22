package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultMaxPoolSize  = 1
	defaultConnAttempts = 10
	defaultConnTimeout  = time.Second
)

var (
	ErrParseConfig = errors.New("failed to parse postgres config")
	ErrCreatePool  = errors.New("failed to create postgres connection pool")
	ErrPing        = errors.New("failed to ping postgres after all attempts")
)

type Postgres struct {
	maxPoolSize  int
	connAttempts int
	connTimeout  time.Duration

	Pool *pgxpool.Pool
}

func New(pgURL string, opts ...Option) (*Postgres, error) {
	pg := &Postgres{
		maxPoolSize:  defaultMaxPoolSize,
		connAttempts: defaultConnAttempts,
		connTimeout:  defaultConnTimeout,
	}

	for _, opt := range opts {
		opt(pg)
	}

	poolConfig, err := pgxpool.ParseConfig(pgURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrParseConfig, err)
	}

	poolConfig.MaxConns = int32(pg.maxPoolSize)

	for pg.connAttempts > 0 {
		pg.Pool, err = pgxpool.NewWithConfig(context.Background(), poolConfig)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrCreatePool, err)
		}

		if err = pg.Pool.Ping(context.Background()); err == nil {
			break
		}

		pg.connAttempts--
		log.Printf("Postgres is not ready, retrying in %s... (%d attempts left)", pg.connTimeout, pg.connAttempts)
		time.Sleep(pg.connTimeout)
	}

	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrPing, err)
	}

	return pg, nil
}

func (p *Postgres) Close() {
	if p.Pool != nil {
		p.Pool.Close()
	}
}
