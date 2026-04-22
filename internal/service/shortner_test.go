package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/Cionaodh/url-shortner-ozon/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock

type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) SaveLink(ctx context.Context, originalURL, shortURL string) (string, error) {
	args := m.Called(ctx, originalURL, shortURL)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) GetLink(ctx context.Context, shortURL string) (string, error) {
	args := m.Called(ctx, shortURL)
	return args.String(0), args.Error(1)
}

// Helpers

func newTestService(storage Storage) *ShortnerService {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	return NewShortnerService(storage, logger)
}

// CreateLink

// Успешный сценарий: короткая ссылка создаётся с первой попытки
func TestCreateLink_Success(t *testing.T) {
	storage := new(MockStorage)
	svc := newTestService(storage)
	ctx := context.Background()

	const originalURL = "https://example.com/very/long/url"
	const savedShortURL = "abc123XYZ_"

	// SaveLink вернёт короткий URL при первом же вызове
	storage.On("SaveLink", ctx, originalURL, mock.AnythingOfType("string")).
		Return(savedShortURL, nil).Once()

	result, err := svc.CreateLink(ctx, originalURL)

	require.NoError(t, err)
	assert.Equal(t, savedShortURL, result)
	storage.AssertExpectations(t)
}

// Сценарий retry: первые 2 попытки — коллизия, третья — успех
func TestCreateLink_CollisionThenSuccess(t *testing.T) {
	storage := new(MockStorage)
	svc := newTestService(storage)
	ctx := context.Background()

	const originalURL = "https://example.com"
	const savedShortURL = "finalURL__"

	storage.On("SaveLink", ctx, originalURL, mock.AnythingOfType("string")).
		Return("", ErrShortURLCollision).Twice()
	storage.On("SaveLink", ctx, originalURL, mock.AnythingOfType("string")).
		Return(savedShortURL, nil).Once()

	result, err := svc.CreateLink(ctx, originalURL)

	require.NoError(t, err)
	assert.Equal(t, savedShortURL, result)
	storage.AssertExpectations(t)
}

// Сценарий исчерпания попыток: все maxRetries попыток — коллизия
func TestCreateLink_AllRetriesCollision(t *testing.T) {
	storage := new(MockStorage)
	svc := newTestService(storage)
	ctx := context.Background()

	// Все MaxRetries попыток — коллизия
	storage.On("SaveLink", ctx, mock.Anything, mock.AnythingOfType("string")).
		Return("", ErrShortURLCollision).Times(maxRetries)

	result, err := svc.CreateLink(ctx, "https://example.com")

	require.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "failed to generate unique short URL")
	storage.AssertExpectations(t)
}

// Сценарий неизвестной ошибки
func TestCreateLink_StorageUnexpectedError(t *testing.T) {
	storage := new(MockStorage)
	svc := newTestService(storage)
	ctx := context.Background()

	unexpectedErr := errors.New("db connection lost")

	storage.On("SaveLink", ctx, mock.Anything, mock.AnythingOfType("string")).
		Return("", unexpectedErr).Once()

	result, err := svc.CreateLink(ctx, "https://example.com")

	require.Error(t, err)
	assert.Empty(t, result)
	// Не должен делать retry при не-коллизионной ошибке
	assert.ErrorIs(t, err, unexpectedErr)
	storage.AssertNumberOfCalls(t, "SaveLink", 1) // ровно 1 вызов, без retry
}

// GetLink

// Успешный сценарий: по короткой ссылке возвращается оригинальный URL
func TestGetLink_Success(t *testing.T) {
	storage := new(MockStorage)
	svc := newTestService(storage)
	ctx := context.Background()

	const shortURL = "abc123XYZ_"
	const originalURL = "https://example.com/very/long/url"

	storage.On("GetLink", ctx, shortURL).Return(originalURL, nil).Once()

	result, err := svc.GetLink(ctx, shortURL)

	require.NoError(t, err)
	assert.Equal(t, originalURL, result)
	storage.AssertExpectations(t)
}

// Сценарий: ссылка не найдена — ErrNotFound
func TestGetLink_NotFound(t *testing.T) {
	storage := new(MockStorage)
	svc := newTestService(storage)
	ctx := context.Background()

	storage.On("GetLink", ctx, "nonexistent").Return("", ErrNotFound).Once()

	result, err := svc.GetLink(ctx, "nonexistent")

	require.Error(t, err)
	assert.Empty(t, result)
	assert.ErrorIs(t, err, ErrNotFound)
	storage.AssertExpectations(t)
}

// Сценарий: неизвестная ошибка хранилища
func TestGetLink_StorageError(t *testing.T) {
	storage := new(MockStorage)
	svc := newTestService(storage)
	ctx := context.Background()

	storageErr := errors.New("timeout")
	storage.On("GetLink", ctx, mock.Anything).Return("", storageErr).Once()

	_, err := svc.GetLink(ctx, "someURL")

	require.Error(t, err)
	assert.ErrorIs(t, err, storageErr)
}

// generateShortURL

// проверка длинны сгенерированной ссылки
func TestGenerateShortURL_Length(t *testing.T) {
	url := generateShortURL()

	assert.Len(t, url, domain.ShortURLLen)
}

// проверка допустимых символов
func TestGenerateShortURL_ValidAlphabet(t *testing.T) {
	for range 100 {
		url := generateShortURL()

		for _, ch := range url {
			assert.True(t,
				strings.ContainsRune(domain.AllowedChars, ch),
				"символ %q не входит в алфавит", ch,
			)
		}
	}
}

// проверка уникальности
func TestGenerateShortURL_Uniqueness(t *testing.T) {
	const iterations = 1000
	seen := make(map[string]struct{}, iterations)

	for range iterations {
		url := generateShortURL()
		seen[url] = struct{}{}
	}

	assert.Len(t, seen, iterations)
}
