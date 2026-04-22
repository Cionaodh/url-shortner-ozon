package httpserver

import (
	"net"
	"time"
)

type Option func(*Server)

// Port задает порт, на котором будет слушать сервер
func Port(port string) Option {
	return func(s *Server) {
		s.HttpServer.Addr = net.JoinHostPort("", port)
	}
}

// ReadTimeout задает максимальное время чтения всего запроса, включая тело
func ReadTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.HttpServer.ReadTimeout = timeout
	}
}

// WriteTimeout задает максимальное время до окончания записи ответа
func WriteTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.HttpServer.WriteTimeout = timeout
	}
}

// ShutdownTimeout задает таймаут для graceful shutdown
func ShutdownTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.shutdownTimeout = timeout
	}
}
