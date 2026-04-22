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

// CreateLink Tests

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

func TestCreateLink_CollisionThenSuccess(t *testing.T) {
	storage := new(MockStorage)
	svc := newTestService(storage)
	ctx := context.Background()

	const originalURL = "https://example.com"
	const savedShortURL = "finalURL__"

	// Первые 2 попытки — коллизия, третья — успех
	storage.On("SaveLink", ctx, originalURL, mock.AnythingOfType("string")).
		Return("", ErrShortURLCollision).Twice()
	storage.On("SaveLink", ctx, originalURL, mock.AnythingOfType("string")).
		Return(savedShortURL, nil).Once()

	result, err := svc.CreateLink(ctx, originalURL)

	require.NoError(t, err)
	assert.Equal(t, savedShortURL, result)
	storage.AssertExpectations(t)
}

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

// GetLink Tests

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

// generateShortURL Unit Tests

func TestGenerateShortURL_Length(t *testing.T) {
	url := generateShortURL()

	assert.Len(t, url, domain.ShortURLLen)
}

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

func TestGenerateShortURL_Uniqueness(t *testing.T) {
	const iterations = 1000
	seen := make(map[string]struct{}, iterations)

	for range iterations {
		url := generateShortURL()
		seen[url] = struct{}{}
	}

	assert.Len(t, seen, iterations)
}

// Fuzz

func FuzzCreateLink_DoesNotPanic(f *testing.F) {
	f.Add("https://example.com")
	f.Add("")
	f.Add("not-a-url")
	f.Add(strings.Repeat("x", 10_000))

	f.Fuzz(func(t *testing.T, originalURL string) {
		storage := new(MockStorage)
		storage.On("SaveLink", mock.Anything, mock.Anything, mock.Anything).
			Return("short123__", nil)

		svc := newTestService(storage)
		// Главное — не паникует при любом входе
		_, _ = svc.CreateLink(context.Background(), originalURL)
	})
}
