package imcache

import (
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"
)

// Элемент кэша
type item[T any] struct {
	value      T
	expiration int64
}

// isExpired проверяет истек ли элемент
func (i item[T]) isExpired() bool {
	if i.expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > i.expiration
}

// shard представляет отдельный шард кэша для уменьшения contention
type shard[T any] struct {
	items map[string]item[T]
	mu    sync.RWMutex
}

// Cache представляет in-memory кэш с шардированием
type Cache[T any] struct {
	shards      []*shard[T]
	shardCount  int
	config      Config[T]
	stopCleanup chan struct{}
	stopWG      sync.WaitGroup
	closed      atomic.Bool
}

// New создает новый экземпляр кэша с опциями
func New[T any](opts ...Option[T]) *Cache[T] {
	config := DefaultConfig[T]()
	for _, opt := range opts {
		opt(&config)
	}

	// Определяем количество шардов
	shardCount := config.ShardCount
	if shardCount <= 0 {
		shardCount = 32 // оптимальное значение по умолчанию
	}

	// Создаем шарды
	shards := make([]*shard[T], shardCount)
	for i := 0; i < shardCount; i++ {
		shards[i] = &shard[T]{
			items: make(map[string]item[T]),
		}
	}

	cache := &Cache[T]{
		shards:      shards,
		shardCount:  shardCount,
		config:      config,
		stopCleanup: make(chan struct{}),
	}

	if config.CleanupInterval > 0 {
		cache.startCleanup()
	}

	return cache
}

// getShard возвращает шард для ключа
func (c *Cache[T]) getShard(key string) *shard[T] {
	if key == "" {
		return c.shards[0]
	}

	h := fnv.New32a()
	h.Write([]byte(key))
	return c.shards[h.Sum32()%uint32(c.shardCount)]
}

// validateKey проверяет валидность ключа
func (c *Cache[T]) validateKey(key string) error {
	if key == "" {
		return ErrInvalidKey
	}
	if c.closed.Load() {
		return ErrCacheClosed
	}
	return nil
}

// Set добавляет значение в кэш
func (c *Cache[T]) Set(key string, value T, ttl ...time.Duration) error {
	if err := c.validateKey(key); err != nil {
		return err
	}

	var expiration int64
	if len(ttl) > 0 && ttl[0] > 0 {
		expiration = time.Now().Add(ttl[0]).UnixNano()
	} else if c.config.DefaultTTL > 0 {
		expiration = time.Now().Add(c.config.DefaultTTL).UnixNano()
	}

	shard := c.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	// Проверяем лимит размера
	if c.config.MaxSize > 0 && len(shard.items) >= c.config.MaxSize/c.shardCount {
		return ErrShardSizeLimit
	}

	shard.items[key] = item[T]{
		value:      value,
		expiration: expiration,
	}

	return nil
}

// Get получает значение из кэша
func (c *Cache[T]) Get(key string) (T, error) {
	var zero T

	if err := c.validateKey(key); err != nil {
		return zero, err
	}

	shard := c.getShard(key)
	shard.mu.RLock()
	item, found := shard.items[key]
	shard.mu.RUnlock()

	if !found {
		if c.config.OnMiss != nil {
			c.config.OnMiss(key)
		}
		return zero, ErrKeyNotFound
	}

	if item.isExpired() {
		// Асинхронное удаление истекшего элемента
		go c.Delete(key)
		if c.config.OnMiss != nil {
			c.config.OnMiss(key)
		}
		return zero, ErrKeyExpired
	}

	if c.config.OnHit != nil {
		c.config.OnHit(key)
	}

	return item.value, nil
}

// Delete удаляет значение из кэша
func (c *Cache[T]) Delete(key string) error {
	if err := c.validateKey(key); err != nil {
		return err
	}

	shard := c.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	item, exists := shard.items[key]
	if exists {
		delete(shard.items, key)
		if c.config.OnEvict != nil {
			c.config.OnEvict(key, item.value)
		}
	}

	return nil
}

// Clear очищает весь кэш
func (c *Cache[T]) Clear() {
	for _, shard := range c.shards {
		shard.mu.Lock()
		if c.config.OnEvict != nil {
			for key, item := range shard.items {
				c.config.OnEvict(key, item.value)
			}
		}
		shard.items = make(map[string]item[T])
		shard.mu.Unlock()
	}
}

// Len возвращает общее количество элементов
func (c *Cache[T]) Len() int {
	count := 0
	for _, shard := range c.shards {
		shard.mu.RLock()
		count += len(shard.items)
		shard.mu.RUnlock()
	}
	return count
}

// Keys возвращает все ключи
func (c *Cache[T]) Keys() []string {
	totalSize := c.Len()
	if totalSize == 0 {
		return []string{}
	}

	keys := make([]string, 0, totalSize)
	for _, shard := range c.shards {
		shard.mu.RLock()
		for key := range shard.items {
			keys = append(keys, key)
		}
		shard.mu.RUnlock()
	}
	return keys
}

// Contains проверяет существование ключа
func (c *Cache[T]) Contains(key string) bool {
	if err := c.validateKey(key); err != nil {
		return false
	}

	shard := c.getShard(key)
	shard.mu.RLock()
	item, found := shard.items[key]
	shard.mu.RUnlock()

	if !found {
		return false
	}
	return !item.isExpired()
}

// TTL возвращает оставшееся время жизни
func (c *Cache[T]) TTL(key string) time.Duration {
	if err := c.validateKey(key); err != nil {
		return -2
	}

	shard := c.getShard(key)
	shard.mu.RLock()
	item, found := shard.items[key]
	shard.mu.RUnlock()

	if !found {
		return -2
	}

	if item.expiration == 0 {
		return -1
	}

	remaining := time.Until(time.Unix(0, item.expiration))
	if remaining < 0 {
		return -2
	}

	return remaining
}

// Expire устанавливает новое время жизни
func (c *Cache[T]) Expire(key string, ttl time.Duration) bool {
	if err := c.validateKey(key); err != nil {
		return false
	}

	shard := c.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	item, found := shard.items[key]
	if !found || item.isExpired() {
		return false
	}

	if ttl > 0 {
		item.expiration = time.Now().Add(ttl).UnixNano()
	} else {
		item.expiration = 0
	}
	shard.items[key] = item
	return true
}

// Range итерируется по всем элементам
func (c *Cache[T]) Range(f func(key string, value T) bool) {
	for _, shard := range c.shards {
		shard.mu.RLock()
		for key, item := range shard.items {
			if !item.isExpired() {
				if !f(key, item.value) {
					shard.mu.RUnlock()
					return
				}
			}
		}
		shard.mu.RUnlock()
	}
}

// startCleanup запускает фоновую очистку
func (c *Cache[T]) startCleanup() {
	c.stopWG.Go(func() {
		ticker := time.NewTicker(c.config.CleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.cleanup()
			case <-c.stopCleanup:
				return
			}
		}
	})
}

// cleanup удаляет истекшие элементы
func (c *Cache[T]) cleanup() {
	if c.closed.Load() {
		return
	}

	now := time.Now().UnixNano()
	evicted := 0

	for _, shard := range c.shards {
		shard.mu.Lock()
		for key, item := range shard.items {
			if item.expiration > 0 && now > item.expiration {
				delete(shard.items, key)
				if c.config.OnEvict != nil {
					c.config.OnEvict(key, item.value)
				}
				evicted++
			}
		}
		shard.mu.Unlock()
	}
}

// Close закрывает кэш и останавливает фоновые процессы
func (c *Cache[T]) Close() error {
	if c.closed.Swap(true) {
		return ErrCacheClosed
	}

	close(c.stopCleanup)
	c.stopWG.Wait()

	c.Clear()

	return nil
}
