package server

type CacheStatus int

const (
	CACHE_MISS    CacheStatus = 1
	CACHE_EXPIRED CacheStatus = 2
	CACHE_SUCCESS CacheStatus = 3
	CACHE_ALREADY CacheStatus = 4
)
