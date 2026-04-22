package httpserver

import (
	"context"
	"errors"
	"net/http"
	"time"
)

const (
	_defaultAddr            = ":80"
	_defaultReadTimeout     = 5 * time.Second
	_defaultWriteTimeout    = 5 * time.Second
	_defaultShutdownTimeout = 3 * time.Second
)

type Server struct {
	HttpServer      *http.Server
	notify          chan error
	shutdownTimeout time.Duration
}

func New(handler http.Handler, opts ...Option) *Server {
	s := &Server{
		HttpServer: &http.Server{
			Addr:         _defaultAddr,
			Handler:      handler,
			ReadTimeout:  _defaultReadTimeout,
			WriteTimeout: _defaultWriteTimeout,
		},
		notify:          make(chan error, 1),
		shutdownTimeout: _defaultShutdownTimeout,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Start запуск сервера в отдельной горутине
func (s *Server) Start() {
	go func() {
		err := s.HttpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.notify <- err
		}
		close(s.notify)
	}()
}

// Notify возврат канала для отслеживания фатальных ошибок сервера
func (s *Server) Notify() <-chan error {
	return s.notify
}

// Shutdown выполняет graceful shutdown сервера
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	return s.HttpServer.Shutdown(ctx)
}
