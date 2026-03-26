package imcache

import "errors"

var (
	// ErrKeyNotFound возвращается когда ключ не найден в кэше
	ErrKeyNotFound = errors.New("imcache: key not found")

	// ErrKeyExpired возвращается когда ключ найден но истек
	ErrKeyExpired = errors.New("imcache: key expired")

	// ErrCacheClosed возвращается при попытке использовать закрытый кэш
	ErrCacheClosed = errors.New("imcache: cache is closed")

	// ErrInvalidTTL возвращается при указании невалидного TTL
	ErrInvalidTTL = errors.New("imcache: invalid ttl")

	// ErrInvalidKey возвращается при указании пустого ключа
	ErrInvalidKey = errors.New("imcache: invalid key")

	// ErrShardSizeLimit возвращается при записи нового ключа в заполненный шард
	ErrShardSizeLimit = errors.New("imcache: cache shard is full")
)
