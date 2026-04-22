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
	defaultMaxPoolSize   = 10
	defaultConnAttempts  = 10
	defaultRetryDelay    = time.Second
	defaultCreateTimeout = 30 * time.Second
	defaultPingTimeout   = 2 * time.Second
)

var (
	ErrParseConfig = errors.New("failed to parse postgres config")
	ErrCreatePool  = errors.New("failed to create postgres connection pool")
	ErrPing        = errors.New("failed to ping postgres after all attempts")
)

type Postgres struct {
	maxPoolSize  int
	connAttempts int
	retryDelay   time.Duration

	Pool *pgxpool.Pool
}

func New(pgURL string, opts ...Option) (*Postgres, error) {
	pg := &Postgres{
		maxPoolSize:  defaultMaxPoolSize,
		connAttempts: defaultConnAttempts,
		retryDelay:   defaultRetryDelay,
	}

	for _, opt := range opts {
		opt(pg)
	}

	poolConfig, err := pgxpool.ParseConfig(pgURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParseConfig, err)
	}

	poolConfig.MaxConns = int32(pg.maxPoolSize)

	ctxCreate, cancelCreate := context.WithTimeout(context.Background(), defaultCreateTimeout)
	defer cancelCreate()

	pg.Pool, err = pgxpool.NewWithConfig(ctxCreate, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCreatePool, err)
	}

	for attemptsLeft := pg.connAttempts; attemptsLeft > 0; attemptsLeft-- {
		ctxPing, cancelPing := context.WithTimeout(context.Background(), defaultPingTimeout)
		err = pg.Pool.Ping(ctxPing)
		cancelPing()

		if err == nil {
			return pg, nil
		}

		remainingAfterThis := attemptsLeft - 1
		log.Printf("Postgres not ready, retrying in %s... (%d attempts left)", pg.retryDelay, remainingAfterThis)

		if remainingAfterThis > 0 {
			time.Sleep(pg.retryDelay)
		}
	}

	pg.Pool.Close()
	return nil, fmt.Errorf("%w: %v", ErrPing, err)
}

func (p *Postgres) Close() {
	if p.Pool != nil {
		p.Pool.Close()
	}
}
