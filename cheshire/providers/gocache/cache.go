package gocache


import (
    cache "github.com/pmylund/go-cache"
    "time"
)

// Wraps github.com/pmylund/go-cache into our local cache interface.


type GoCache struct {
    Cache *cache.Cache
}

// Creates a new GoCache with the given intervals
// defaultExpiration = 0 is never expire
// cleanupInterval = 0 is never attempt to clean up expired 
func New(defaultExpirationSeconds, cleanupIntervalSeconds int) *GoCache {
    return &GoCache{
        Cache : cache.New(time.Duration(defaultExpirationSeconds) * time.Second, time.Duration(cleanupIntervalSeconds) * time.Second),
    }
}

func (this *GoCache) Set(key string, value []byte, expireSeconds int) {
    this.Cache.Set(key, value, time.Duration(expireSeconds)*time.Second)
}


    // Sets the value if and only if there is no value associated with this key
func (this *GoCache) SetIfAbsent(key string, value []byte, expireSeconds int) bool {
    err := this.Cache.Add(key, value, time.Duration(expireSeconds)*time.Second)
    return err == nil
}
    
    // Deletes the value at the requested key
func (this *GoCache) Delete(key string) bool {
    this.Cache.Delete(key)
}

    // Gets the value at the requested key
func (this *GoCache) Get(key string) ([]byte, bool) {
    v, ok := this.Cache.Get(key)
    if !ok {
        return make([]byte,0), false
    }
    bt, ok := v.([]byte)
    return bt, ok
}

    // // Increment the key by val (val is allowed to be negative)
    // // in most implementation expireSeconds will be from the first increment, but users should not count on that.
    // // if no value is a present it should be added.  
    // // If a value is present which is not a number an error should be returned.
    // Inc(key string, val int64, expireSeconds int) (int64, error)