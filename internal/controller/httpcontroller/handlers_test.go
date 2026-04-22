package httpcontroller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Cionaodh/url-shortner-ozon/internal/service"
)

// Mock

type MockService struct {
	mock.Mock
}

func (m *MockService) CreateLink(ctx context.Context, url string) (string, error) {
	args := m.Called(ctx, url)
	return args.String(0), args.Error(1)
}

func (m *MockService) GetLink(ctx context.Context, shortURL string) (string, error) {
	args := m.Called(ctx, shortURL)
	return args.String(0), args.Error(1)
}

// Helpers

func newTestRouter(svc Service) http.Handler {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	return NewRouter(svc, logger)
}

func makeCreateRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func makeGetRequest(short string) *http.Request {
	return httptest.NewRequest(http.MethodGet, "/"+short, nil)
}

func decodeError(t *testing.T, body *bytes.Buffer) ErrorResp {
	t.Helper()
	var resp ErrorResp
	require.NoError(t, json.NewDecoder(body).Decode(&resp))
	return resp
}

// CreateLink

func TestCreateLink_Success(t *testing.T) {
	svc := new(MockService)
	svc.On("CreateLink", mock.Anything, "https://example.com").
		Return("abc123XYZ_", nil).Once()

	rec := httptest.NewRecorder()
	newTestRouter(svc).ServeHTTP(rec, makeCreateRequest(`{"url":"https://example.com"}`))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp ShortURL
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "abc123XYZ_", resp.ShortURL)

	svc.AssertExpectations(t)
}

func TestCreateLink_InvalidJSON(t *testing.T) {
	svc := new(MockService)
	rec := httptest.NewRecorder()

	newTestRouter(svc).ServeHTTP(rec, makeCreateRequest(`not json`))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid request", decodeError(t, rec.Body).Error)
	svc.AssertNotCalled(t, "CreateLink")
}

func TestCreateLink_EmptyURL(t *testing.T) {
	svc := new(MockService)
	rec := httptest.NewRecorder()

	newTestRouter(svc).ServeHTTP(rec, makeCreateRequest(`{"url":""}`))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid url", decodeError(t, rec.Body).Error)
	svc.AssertNotCalled(t, "CreateLink")
}

func TestCreateLink_URLTooLong(t *testing.T) {
	svc := new(MockService)
	rec := httptest.NewRecorder()

	longURL := "https://example.com/" + strings.Repeat("a", maxURLLen)
	body, _ := json.Marshal(OriginULR{URL: longURL})

	newTestRouter(svc).ServeHTTP(rec, makeCreateRequest(string(body)))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid url", decodeError(t, rec.Body).Error)
	svc.AssertNotCalled(t, "CreateLink")
}

func TestCreateLink_InvalidURLFormat(t *testing.T) {
	cases := []struct {
		name string
		url  string
	}{
		{"no host", "https://"},
		{"just text", "not-a-url"},
		{"no tld", "https://localhost"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := new(MockService)
			rec := httptest.NewRecorder()

			body, _ := json.Marshal(OriginULR{URL: tc.url})
			newTestRouter(svc).ServeHTTP(rec, makeCreateRequest(string(body)))

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			svc.AssertNotCalled(t, "CreateLink")
		})
	}
}

func TestCreateLink_ServiceError(t *testing.T) {
	svc := new(MockService)
	svc.On("CreateLink", mock.Anything, mock.Anything).
		Return("", errors.New("unexpected db error")).Once()

	rec := httptest.NewRecorder()
	newTestRouter(svc).ServeHTTP(rec, makeCreateRequest(`{"url":"https://example.com"}`))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "internal server error", decodeError(t, rec.Body).Error)
	svc.AssertExpectations(t)
}

// GetLink

func TestGetLink_Success(t *testing.T) {
	svc := new(MockService)
	svc.On("GetLink", mock.Anything, "abc123XYZ_").
		Return("https://example.com", nil).Once()

	rec := httptest.NewRecorder()
	newTestRouter(svc).ServeHTTP(rec, makeGetRequest("abc123XYZ_"))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp OriginULR
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "https://example.com", resp.URL)

	svc.AssertExpectations(t)
}

func TestGetLink_ShortURLWrongLength(t *testing.T) {
	svc := new(MockService)
	rec := httptest.NewRecorder()

	newTestRouter(svc).ServeHTTP(rec, makeGetRequest("short"))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid short link", decodeError(t, rec.Body).Error)
	svc.AssertNotCalled(t, "GetLink")
}

func TestGetLink_ShortURLInvalidChars(t *testing.T) {
	svc := new(MockService)
	rec := httptest.NewRecorder()

	// 10 символов, но с недопустимыми символами
	newTestRouter(svc).ServeHTTP(rec, makeGetRequest("abc!@#XYZ_"))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid short link", decodeError(t, rec.Body).Error)
	svc.AssertNotCalled(t, "GetLink")
}

func TestGetLink_NotFound(t *testing.T) {
	svc := new(MockService)
	svc.On("GetLink", mock.Anything, "abc123XYZ_").
		Return("", service.ErrNotFound).Once()

	rec := httptest.NewRecorder()
	newTestRouter(svc).ServeHTTP(rec, makeGetRequest("abc123XYZ_"))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "not found", decodeError(t, rec.Body).Error)
	svc.AssertExpectations(t)
}

func TestGetLink_ServiceError(t *testing.T) {
	svc := new(MockService)
	svc.On("GetLink", mock.Anything, "abc123XYZ_").
		Return("", errors.New("db timeout")).Once()

	rec := httptest.NewRecorder()
	newTestRouter(svc).ServeHTTP(rec, makeGetRequest("abc123XYZ_"))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "internal server error", decodeError(t, rec.Body).Error)
	svc.AssertExpectations(t)
}

// validateURL

func TestValidateURL(t *testing.T) {
	cases := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid https", "https://example.com", false},
		{"valid http", "http://example.com/path?q=1", false},
		{"without scheme", "example.com", false}, // нормализуется до https://
		{"empty", "", true},
		{"too long", strings.Repeat("a", maxURLLen+1), true},
		{"no host", "https://", true},
		{"only spaces", "   ", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateURL(tc.url)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// validateShortURL

func TestValidateShortURL(t *testing.T) {
	cases := []struct {
		name    string
		short   string
		wantErr bool
	}{
		{"valid", "abc123XYZ_", false},
		{"too short", "abc", true},
		{"too long", "abc123XYZ_extra", true},
		{"invalid char space", "abc123XY Z", true},
		{"invalid char dash", "abc123XYZ-", true},
		{"all underscores", "__________", false},
		{"empty", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateShortURL(tc.short)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Fuzz

func FuzzCreateLink_DoesNotPanic(f *testing.F) {
	f.Add(`{"url":"https://example.com"}`)
	f.Add(`{"url":""}`)
	f.Add(`not json`)
	f.Add(`{"url":"` + strings.Repeat("a", 10_000) + `"}`)

	f.Fuzz(func(t *testing.T, body string) {
		svc := new(MockService)
		svc.On("CreateLink", mock.Anything, mock.Anything).
			Return("abc123XYZ_", nil)

		rec := httptest.NewRecorder()
		newTestRouter(svc).ServeHTTP(rec, makeCreateRequest(body))

		// Любой ввод — только валидные HTTP-статусы
		assert.Contains(t,
			[]int{http.StatusOK, http.StatusBadRequest, http.StatusInternalServerError},
			rec.Code,
		)
	})
}

func FuzzValidateURL_DoesNotPanic(f *testing.F) {
	f.Add("https://example.com")
	f.Add("")
	f.Add(strings.Repeat("x", 10_000))

	f.Fuzz(func(t *testing.T, rawURL string) {
		_ = validateURL(rawURL)
	})
}

// errorResponse

func TestErrorResponse(t *testing.T) {
	cases := []struct {
		name   string
		msg    string
		status int
	}{
		{"bad request", "invalid request", http.StatusBadRequest},
		{"not found", "not found", http.StatusNotFound},
		{"internal error", "internal server error", http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			errorResponse(rec, tc.msg, tc.status)

			assert.Equal(t, tc.status, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			var resp ErrorResp
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
			assert.Equal(t, tc.msg, resp.Error)
		})
	}
}
