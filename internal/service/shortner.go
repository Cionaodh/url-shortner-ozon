package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Cionaodh/url-shortner-ozon/internal/domain"
)

var (
	ErrShortURLCollision = errors.New("short url collision")
	ErrNotFound          = errors.New("url not found")
)

type Storage interface {
	SaveLink(ctx context.Context, originalURL, shortURL string) (string, error)
	GetLink(ctx context.Context, shortURL string) (string, error)
}

const (
	maxRetries = 5
)

type ShortnerService struct {
	repo Storage
	l    *slog.Logger
}

func NewShortnerService(rp Storage, log *slog.Logger) *ShortnerService {
	return &ShortnerService{
		repo: rp,
		l:    log.With("component", "service"),
	}
}

// CreateLink создает и сохраняет короткую ссылку для переданного URL
func (s *ShortnerService) CreateLink(ctx context.Context, originalURL string) (string, error) {
	for i := 0; i < maxRetries; i++ {
		shortURL := generateShortURL()

		savedShortURL, err := s.repo.SaveLink(ctx, originalURL, shortURL)
		if err != nil {
			// ретраим колизию
			if errors.Is(err, ErrShortURLCollision) {
				s.l.Warn("collision detected, retrying", "attempt", i+1, "shortURL", shortURL)
				continue
			}

			return "", fmt.Errorf("failed to save link: %w", err)
		}

		return savedShortURL, nil
	}

	return "", fmt.Errorf("failed to generate unique short URL after %d attempts", maxRetries)
}

// GetLink получение из короткой ссылки - оригинальной ссылки
func (s *ShortnerService) GetLink(ctx context.Context, shortURL string) (string, error) {
	originURL, err := s.repo.GetLink(ctx, shortURL)
	if err != nil {
		return "", fmt.Errorf("failed to get link: %w", err)
	}
	return originURL, nil
}

func generateShortURL() string {
	bytes := make([]byte, domain.ShortURLLen)

	rand.Read(bytes)

	for i, b := range bytes {
		bytes[i] = domain.AllowedChars[int(b)%len(domain.AllowedChars)]
	}

	return string(bytes)
}
