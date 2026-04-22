package pgdb_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Cionaodh/url-shortner-ozon/internal/repo/pgdb"
	"github.com/Cionaodh/url-shortner-ozon/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Моки

type mockRow struct {
	scanFunc func(dest ...any) error
}

func (m *mockRow) Scan(dest ...any) error {
	if m.scanFunc == nil {
		return errors.New("scanFunc not set")
	}
	return m.scanFunc(dest...)
}

type mockQuerier struct {
	row pgx.Row
}

func (m *mockQuerier) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return m.row
}

// Хелпер

func newStorage(row pgx.Row) *pgdb.Storage {
	return pgdb.NewStorage(&mockQuerier{row: row})
}

// SaveLink

func TestSaveLink_Success(t *testing.T) {
	row := &mockRow{
		scanFunc: func(dest ...any) error {
			*dest[0].(*string) = "abc123"
			return nil
		},
	}

	got, err := newStorage(row).SaveLink(context.Background(), "https://example.com", "abc123")

	require.NoError(t, err)
	assert.Equal(t, "abc123", got)
}

func TestSaveLink_Collision(t *testing.T) {
	row := &mockRow{
		scanFunc: func(dest ...any) error {
			return &pgconn.PgError{
				Code:           "23505",
				ConstraintName: "links_short_url_key",
			}
		},
	}

	_, err := newStorage(row).SaveLink(context.Background(), "https://example.com", "abc123")

	assert.ErrorIs(t, err, service.ErrShortURLCollision)
}

func TestSaveLink_UnexpectedError(t *testing.T) {
	row := &mockRow{
		scanFunc: func(dest ...any) error {
			return errors.New("unexpected db error")
		},
	}

	_, err := newStorage(row).SaveLink(context.Background(), "https://example.com", "abc123")

	require.Error(t, err)
}

// GetLink

func TestGetLink_Success(t *testing.T) {
	row := &mockRow{
		scanFunc: func(dest ...any) error {
			*dest[0].(*string) = "https://example.com"
			return nil
		},
	}

	got, err := newStorage(row).GetLink(context.Background(), "abc123")

	require.NoError(t, err)
	assert.Equal(t, "https://example.com", got)
}

func TestGetLink_NotFound(t *testing.T) {
	row := &mockRow{
		scanFunc: func(dest ...any) error {
			return pgx.ErrNoRows
		},
	}

	_, err := newStorage(row).GetLink(context.Background(), "abc123")

	assert.ErrorIs(t, err, service.ErrNotFound)
}

func TestGetLink_UnexpectedError(t *testing.T) {
	row := &mockRow{
		scanFunc: func(dest ...any) error {
			return errors.New("unexpected db error")
		},
	}

	_, err := newStorage(row).GetLink(context.Background(), "abc123")

	require.Error(t, err)
}
