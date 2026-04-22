package httpcontroller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/Cionaodh/url-shortner-ozon/internal/domain"
	"github.com/Cionaodh/url-shortner-ozon/internal/service"
	"github.com/go-chi/chi/v5"
)

const (
	maxURLLen = 2000
)

type Service interface {
	CreateLink(ctx context.Context, url string) (string, error)
	GetLink(ctx context.Context, shortURL string) (string, error)
}

type Handler struct {
	s   Service
	log *slog.Logger
}

func NewHandler(s Service, log *slog.Logger) *Handler {
	return &Handler{s: s, log: log}
}

type OriginULR struct {
	URL string `json:"url"`
}

type ShortURL struct {
	ShortURL string `json:"short_url"`
}

func (h *Handler) CreateLink(w http.ResponseWriter, r *http.Request) {
	var req OriginULR
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, "invalid request", http.StatusBadRequest)
		return
	}

	if err := validateURL(req.URL); err != nil {
		errorResponse(w, "invalid url", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	short, err := h.s.CreateLink(ctx, req.URL)
	if err != nil {
		errorResponse(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ShortURL{ShortURL: short})
}

func (h *Handler) GetLink(w http.ResponseWriter, r *http.Request) {
	short := chi.URLParam(r, "short")

	if err := validateShortURL(short); err != nil {
		errorResponse(w, "invalid short link", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	original, err := h.s.GetLink(ctx, short)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			errorResponse(w, "not found", http.StatusNotFound)
			return
		}
		errorResponse(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(OriginULR{URL: original})
}

func validateURL(rawURL string) error {
	if rawURL == "" {
		return errors.New("url cannot be empty")
	}

	if utf8.RuneCountInString(rawURL) > maxURLLen {
		return fmt.Errorf("url is too long")
	}

	normalized := rawURL
	if !strings.Contains(rawURL, "://") {
		normalized = "https://" + rawURL
	}

	parsed, err := url.ParseRequestURI(normalized)
	if err != nil || parsed.Host == "" {
		return errors.New("url has an incorrect format")
	}

	if !strings.Contains(parsed.Host, ".") {
		return errors.New("url has an incorrect format")
	}

	return nil
}

func validateShortURL(short string) error {
	if len(short) != domain.ShortURLLen {
		return fmt.Errorf("url code must consist of %d characters", domain.ShortURLLen)
	}

	for _, ch := range short {
		if !strings.ContainsRune(domain.AllowedChars, ch) {
			return fmt.Errorf("invalid characters")
		}
	}

	return nil
}
