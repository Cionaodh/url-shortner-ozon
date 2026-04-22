package postgres

import "time"

type Option func(*Postgres)

// MaxPoolSize устанавливает максимальное количество соединений в пуле
func MaxPoolSize(size int) Option {
	return func(c *Postgres) {
		c.maxPoolSize = size
	}
}

// ConnAttempts устанавливает количество попыток подключения к базе данных
func ConnAttempts(attempts int) Option {
	return func(c *Postgres) {
		c.connAttempts = attempts
	}
}

// ConnTimeout устанавливает время ожидания между попытками подключения
func ConnTimeout(timeout time.Duration) Option {
	return func(c *Postgres) {
		c.connTimeout = timeout
	}
}
