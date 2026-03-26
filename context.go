package imcache

import "context"

// WithContext создает контекстный кэш с поддержкой отмены
type CacheWithContext[T any] struct {
	*Cache[T]
	ctx context.Context
}

// NewWithContext создает кэш с контекстом
func NewWithContext[T any](ctx context.Context, opts ...Option[T]) *CacheWithContext[T] {
	cache := New(opts...)

	go func() {
		<-ctx.Done()
		cache.Close()
	}()

	return &CacheWithContext[T]{
		Cache: cache,
		ctx:   ctx,
	}
}
