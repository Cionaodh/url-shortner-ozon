package inmemory

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/Cionaodh/url-shortner-ozon/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

// SaveLink

func TestSaveLink_Success(t *testing.T) {
	s := NewStorage()

	short, err := s.SaveLink(ctx, "https://example.com", "abc123XYZ_")

	require.NoError(t, err)
	assert.Equal(t, "abc123XYZ_", short)
}

func TestSaveLink_DuplicateOriginURL_ReturnExistingShort(t *testing.T) {
	s := NewStorage()

	short1, err := s.SaveLink(ctx, "https://example.com", "abc123XYZ_")
	require.NoError(t, err)

	// Та же оригинальная ссылка — должна вернуть уже существующий короткий код
	short2, err := s.SaveLink(ctx, "https://example.com", "newShortXYZ")

	require.NoError(t, err)
	assert.Equal(t, short1, short2)
}

func TestSaveLink_Collision_ReturnError(t *testing.T) {
	s := NewStorage()

	_, err := s.SaveLink(ctx, "https://example.com", "abc123XYZ_")
	require.NoError(t, err)

	// Другой оригинальный URL, но тот же короткий код — коллизия
	_, err = s.SaveLink(ctx, "https://other.com", "abc123XYZ_")

	assert.ErrorIs(t, err, service.ErrShortURLCollision)
}

func TestSaveLink_DifferentURLs_StoredIndependently(t *testing.T) {
	s := NewStorage()

	short1, err := s.SaveLink(ctx, "https://example.com", "aaaaaaaaaa")
	require.NoError(t, err)

	short2, err := s.SaveLink(ctx, "https://other.com", "bbbbbbbbbb")
	require.NoError(t, err)

	assert.Equal(t, "aaaaaaaaaa", short1)
	assert.Equal(t, "bbbbbbbbbb", short2)
}

// GetLink

func TestGetLink_Success(t *testing.T) {
	s := NewStorage()

	_, err := s.SaveLink(ctx, "https://example.com", "abc123XYZ_")
	require.NoError(t, err)

	original, err := s.GetLink(ctx, "abc123XYZ_")

	require.NoError(t, err)
	assert.Equal(t, "https://example.com", original)
}

func TestGetLink_NotFound_ReturnError(t *testing.T) {
	s := NewStorage()

	_, err := s.GetLink(ctx, "notexists_1")

	assert.ErrorIs(t, err, service.ErrNotFound)
}

func TestGetLink_AfterSave_ReturnsCorrectOriginal(t *testing.T) {
	s := NewStorage()

	urls := map[string]string{
		"aaaaaaaaaa": "https://first.com",
		"bbbbbbbbbb": "https://second.com",
		"cccccccccc": "https://third.com",
	}

	for short, original := range urls {
		_, err := s.SaveLink(ctx, original, short)
		require.NoError(t, err)
	}

	for short, expected := range urls {
		got, err := s.GetLink(ctx, short)
		require.NoError(t, err)
		assert.Equal(t, expected, got)
	}
}

// Concurrency

func TestStorage_ConcurrentSave_NoDataRace(t *testing.T) {
	s := NewStorage()
	const goroutines = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(i int) {
			defer wg.Done()
			short := fmt.Sprintf("short%05d", i)
			original := fmt.Sprintf("https://example%d.com", i)
			_, _ = s.SaveLink(ctx, original, short)
		}(i)
	}

	wg.Wait()
}

func TestStorage_ConcurrentReadWrite_NoDataRace(t *testing.T) {
	s := NewStorage()

	// Предзаполняем хранилище
	_, err := s.SaveLink(ctx, "https://example.com", "abc123XYZ_")
	require.NoError(t, err)

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// Параллельные чтения
	for range goroutines {
		go func() {
			defer wg.Done()
			_, _ = s.GetLink(ctx, "abc123XYZ_")
		}()
	}

	// Параллельные записи
	for i := range goroutines {
		go func(i int) {
			defer wg.Done()
			short := fmt.Sprintf("short%05d", i)
			original := fmt.Sprintf("https://site%d.com", i)
			_, _ = s.SaveLink(ctx, original, short)
		}(i)
	}

	wg.Wait()
}
