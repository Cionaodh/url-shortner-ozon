package pgdb

import (
	"context"
	"errors"

	"github.com/Cionaodh/url-shortner-ozon/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Storage struct {
	pool Querier
}

func NewStorage(pg Querier) *Storage {
	return &Storage{pg}
}

// SaveLink сохраняет ссылку или возвращает существующую
func (st *Storage) SaveLink(ctx context.Context, originalURL, shortURL string) (string, error) {
	// SET origin_url = links.origin_url - является костылем, чтобы срабатывал RETURNING при повторах оригинального url
	// по сути перезаписываем ту же строку
	// но данная конструкция позволяет добиться атомарности операции
	query := `
		INSERT INTO links (origin_url, short_url) 
		VALUES ($1, $2) 
		ON CONFLICT (origin_url) DO UPDATE 
		SET origin_url = links.origin_url
		RETURNING short_url;
	`

	var returnedShortURL string
	err := st.pool.QueryRow(ctx, query, originalURL, shortURL).Scan(&returnedShortURL)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "links_short_url_key" {
			// 23505 - код ошибки unique_violation - произошла колизия
			return "", service.ErrShortURLCollision
		}
		return "", err
	}

	return returnedShortURL, nil
}

func (st *Storage) GetLink(ctx context.Context, shortURL string) (string, error) {
	query := `
		SELECT origin_url FROM links
		WHERE short_url = $1;
	`

	var originURL string
	err := st.pool.QueryRow(ctx, query, shortURL).Scan(&originURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", service.ErrNotFound
		}
		return "", err
	}

	return originURL, nil
}
