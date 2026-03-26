package imcache

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCache_BasicOperations(t *testing.T) {
	cache := New[string]()
	defer cache.Close()

	// Тестовая запись и получение
	err := cache.Set("key1", "value1")
	assert.NoError(t, err)

	val, err := cache.Get("key1")
	assert.NoError(t, err)
	assert.Equal(t, "value1", val)

	// Проверка получения несуществующего ключа
	_, err = cache.Get("nonexistent")
	assert.ErrorIs(t, err, ErrKeyNotFound)

	// Тестовое удаление
	err = cache.Delete("key1")
	assert.NoError(t, err)

	_, err = cache.Get("key1")
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestCache_TTL(t *testing.T) {
	cache := New[string]()
	defer cache.Close()

	// Тест с TTL
	err := cache.Set("temp", "value", 100*time.Millisecond)
	assert.NoError(t, err)

	val, err := cache.Get("temp")
	assert.NoError(t, err)
	assert.Equal(t, "value", val)

	// Дождемся истечения срока действия
	time.Sleep(150 * time.Millisecond)

	_, err = cache.Get("temp")
	assert.ErrorIs(t, err, ErrKeyExpired)
}

func TestCache_ConcurrentAccess(t *testing.T) {
	cache := New[int]()
	defer cache.Close()

	var wg sync.WaitGroup
	iterations := 1000

	// Параллельная запись
	for i := range iterations {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cache.Set("counter", i)
		}(i)
	}

	// Параллельное чтение
	for range iterations {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cache.Get("counter")
		}()
	}

	wg.Wait()

	// Проверяем работу кэша
	_, err := cache.Get("counter")
	assert.NoError(t, err)
}

func TestCache_Cleanup(t *testing.T) {
	cache := New(
		WithCleanupInterval[string](140 * time.Millisecond),
	)
	defer cache.Close()

	cache.Set("expiring", "value", 30*time.Millisecond)
	cache.Set("permanent", "value", 0)

	// Дождемся очистки
	time.Sleep(100 * time.Millisecond)

	_, err := cache.Get("expiring")
	assert.ErrorIs(t, err, ErrKeyExpired)

	_, err = cache.Get("permanent")
	assert.NoError(t, err)
}

func TestCache_EvictCallback(t *testing.T) {
	evicted := make([]string, 0)
	var mu sync.Mutex

	cache := New(
		WithOnEvict(func(key string, value string) {
			mu.Lock()
			evicted = append(evicted, key)
			mu.Unlock()
		}),
	)
	defer cache.Close()

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	cache.Delete("key1")
	cache.Delete("key2")

	mu.Lock()
	assert.ElementsMatch(t, []string{"key1", "key2"}, evicted)
	mu.Unlock()
}

func TestCache_MaxSize(t *testing.T) {
	cache := New(
		WithMaxSize[int](10),
	)
	defer cache.Close()

	// Заполним кэш
	for i := range 10 {
		err := cache.Set(string(rune('a'+i)), i)
		assert.ErrorIs(t, err, ErrShardSizeLimit)
	}

	// Это вызовет ошибку
	err := cache.Set("overflow", 999)
	assert.Error(t, err)
}

func TestCache_Close(t *testing.T) {
	cache := New[string]()

	err := cache.Close()
	assert.NoError(t, err)

	// Операция должна провалиться после закрытия
	err = cache.Set("key", "value")
	assert.ErrorIs(t, err, ErrCacheClosed)

	_, err = cache.Get("key")
	assert.ErrorIs(t, err, ErrCacheClosed)
}

// Benchmark tests
func BenchmarkCache_Set(b *testing.B) {
	cache := New[string]()
	defer cache.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cache.Set("key", "value")
			i++
		}
	})
}

func BenchmarkCache_Get(b *testing.B) {
	cache := New[string]()
	cache.Set("key", "value")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache.Get("key")
		}
	})
}

func BenchmarkCache_ConcurrentReadWrite(b *testing.B) {
	cache := New[string]()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				cache.Set("key", "value")
			} else {
				cache.Get("key")
			}
			i++
		}
	})
}
