package server

import (
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

type AuthCacheType struct {
	userid        int
	date_creation time.Time
}

type AuthCache struct {
	cache   map[uuid.UUID]AuthCacheType
	mtx     sync.RWMutex
	timeout int
}

func NewAuthCache(Timeout int) *AuthCache {
  return  &AuthCache{
		cache:   make(map[uuid.UUID]AuthCacheType),
		mtx:     sync.RWMutex{},
		timeout: Timeout,
	}
}

func (a *AuthCache) Get(k uuid.UUID) CacheStatus {
	if a == nil {
		return CACHE_MISS
	}
	a.mtx.RLock()
	f, ok := a.cache[k];
  a.mtx.RUnlock()
  if ok {
		if int(time.Since(f.date_creation).Minutes()) < a.timeout {
			return CACHE_SUCCESS
		}
		a.mtx.Lock()
		a.Remove(k)
		a.mtx.Unlock()
		return CACHE_EXPIRED
	}
  
	return CACHE_MISS
}

func (a *AuthCache) GetUserUUID(k uuid.UUID) *int {
	a.mtx.RLock()
	v, ok := a.cache[k]
	a.mtx.RUnlock()
	if ok {
    log.Println("GetUserUUID: userid"+ strconv.Itoa(v.userid))
		return &v.userid
	}

	return nil
}

func (a *AuthCache) Remove(k uuid.UUID) CacheStatus {
	a.mtx.Lock()
	if _, ok := a.cache[k]; ok {
		delete(a.cache, k)
		a.mtx.Unlock()
		return CACHE_SUCCESS
	}
	a.mtx.Unlock()
	return CACHE_MISS
}

func (a *AuthCache) Set(k uuid.UUID, act AuthCacheType) CacheStatus {
	if a.Get(k) == CACHE_SUCCESS {
		return CACHE_ALREADY
	}
	a.mtx.Lock()
	a.cache[k] = act
	a.mtx.Unlock()
	return CACHE_SUCCESS
}
