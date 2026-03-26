package imcache

import "time"

// Option определяет функциональную опцию для конфигурации кэша
type Option[T any] func(*Config[T])

// Config содержит конфигурацию кэша
type Config[T any] struct {
	// DefaultTTL - TTL по умолчанию для новых записей (0 = бесконечно)
	DefaultTTL time.Duration

	// CleanupInterval - интервал автоматической очистки (0 = отключено)
	CleanupInterval time.Duration

	// OnEvict - колбэк вызываемый при удалении записи
	OnEvict func(key string, value T)

	// OnMiss - колбэк вызываемый при промахе кэша
	OnMiss func(key string)

	// OnHit - колбэк вызываемый при попадании в кэш
	OnHit func(key string)

	// EnableStats - включить сбор статистики
	EnableStats bool

	// MaxSize - максимальное количество записей (0 = без ограничений)
	MaxSize int

	// ShardCount - количество шардов для уменьшения блокировок (0 = авто)
	ShardCount int

	// EnableLogging - включить логирование
	EnableLogging bool
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig[T any]() Config[T] {
	return Config[T]{
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		EnableStats:     false,
		MaxSize:         0,
		ShardCount:      0,
		EnableLogging:   false,
	}
}

// WithDefaultTTL устанавливает TTL по умолчанию
func WithDefaultTTL[T any](ttl time.Duration) Option[T] {
	return func(c *Config[T]) {
		if ttl >= 0 {
			c.DefaultTTL = ttl
		}
	}
}

// WithCleanupInterval устанавливает интервал очистки
func WithCleanupInterval[T any](interval time.Duration) Option[T] {
	return func(c *Config[T]) {
		if interval > 0 {
			c.CleanupInterval = interval
		}
	}
}

// WithOnEvict устанавливает колбэк при удалении
func WithOnEvict[T any](fn func(key string, value T)) Option[T] {
	return func(c *Config[T]) {
		c.OnEvict = fn
	}
}

// WithOnMiss устанавливает колбэк при промахе
func WithOnMiss[T any](fn func(key string)) Option[T] {
	return func(c *Config[T]) {
		c.OnMiss = fn
	}
}

// WithOnHit устанавливает колбэк при попадании
func WithOnHit[T any](fn func(key string)) Option[T] {
	return func(c *Config[T]) {
		c.OnHit = fn
	}
}

// WithMaxSize устанавливает максимальный размер кэша
func WithMaxSize[T any](size int) Option[T] {
	return func(c *Config[T]) {
		if size > 0 {
			c.MaxSize = size
		}
	}
}

// WithShards устанавливает количество шардов
func WithShards[T any](count int) Option[T] {
	return func(c *Config[T]) {
		if count > 0 {
			c.ShardCount = count
		}
	}
}
