package inmemory

import (
	"context"
	"sync"

	"github.com/Cionaodh/url-shortner-ozon/internal/service"
)

type Storage struct {
	mu       sync.RWMutex
	toOrigin map[string]string
	toShort  map[string]string
}

func NewStorage() *Storage {
	return &Storage{
		toOrigin: make(map[string]string),
		toShort:  make(map[string]string),
	}
}

func (s *Storage) SaveLink(_ context.Context, originURL, shortURL string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// ссылка уже записана в базе
	if existShort, ok := s.toShort[originURL]; ok {
		return existShort, nil
	}

	// коллизия - короткая ссылка принадлежит другому оригинальному url
	if _, ok := s.toOrigin[shortURL]; ok {
		return "", service.ErrShortURLCollision
	}

	s.toOrigin[shortURL] = originURL
	s.toShort[originURL] = shortURL

	return shortURL, nil
}

func (s *Storage) GetLink(_ context.Context, shortURL string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	originURL, ok := s.toOrigin[shortURL]
	if !ok {
		return "", service.ErrNotFound
	}

	return originURL, nil
}
