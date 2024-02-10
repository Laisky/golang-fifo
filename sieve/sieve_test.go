package sieve

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const noEvictionTTL = 0

func TestGetAndSet(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	cache := New[int, int](10, noEvictionTTL)

	for _, v := range items {
		cache.Set(v, v*10)
	}

	for _, v := range items {
		val, ok := cache.Get(v)
		require.True(t, ok)
		require.Equal(t, v*10, val)
	}

	cache.Close()
}

func TestRemove(t *testing.T) {
	cache := New[int, int](10, noEvictionTTL)
	cache.Set(1, 10)

	val, ok := cache.Get(1)
	require.True(t, ok)
	require.Equal(t, 10, val)

	// After removing the key, it should not be found
	removed := cache.Remove(1)
	require.True(t, removed)

	_, ok = cache.Get(1)
	require.False(t, ok)

	// This should not panic
	removed = cache.Remove(-1)
	require.False(t, removed)

	cache.Close()
}

func TestSievePolicy(t *testing.T) {
	cache := New[int, int](10, noEvictionTTL)
	oneHitWonders := []int{1, 2, 3, 4, 5}
	popularObjects := []int{6, 7, 8, 9, 10}

	// add objects to the cache
	for _, v := range oneHitWonders {
		cache.Set(v, v)
	}
	for _, v := range popularObjects {
		cache.Set(v, v)
	}

	// hit popular objects
	for _, v := range popularObjects {
		_, ok := cache.Get(v)
		require.True(t, ok)
	}

	// add another objects to the cache
	for _, v := range oneHitWonders {
		cache.Set(v*10, v*10)
	}

	// check popular objects are not evicted
	for _, v := range popularObjects {
		_, ok := cache.Get(v)
		require.True(t, ok)
	}

	cache.Close()
}

func TestContains(t *testing.T) {
	cache := New[string, string](10, noEvictionTTL)
	require.False(t, cache.Contains("hello"))

	cache.Set("hello", "world")
	require.True(t, cache.Contains("hello"))

	cache.Close()
}

func TestLen(t *testing.T) {
	cache := New[int, int](10, noEvictionTTL)
	require.Equal(t, 0, cache.Len())

	cache.Set(1, 1)
	require.Equal(t, 1, cache.Len())

	// duplicated keys only update the recent-ness of the key and value
	cache.Set(1, 1)
	require.Equal(t, 1, cache.Len())

	cache.Set(2, 2)
	require.Equal(t, 2, cache.Len())

	cache.Close()
}

func TestPurge(t *testing.T) {
	cache := New[int, int](10, noEvictionTTL)
	cache.Set(1, 1)
	cache.Set(2, 2)
	require.Equal(t, 2, cache.Len())

	cache.Purge()
	require.Equal(t, 0, cache.Len())

	cache.Close()
}

func TestTimeToLive(t *testing.T) {
	ttl := time.Second
	cache := New[int, int](10, ttl)
	numberOfEntries := 10

	for num := 1; num <= numberOfEntries; num++ {
		cache.Set(num, num)
		val, ok := cache.Get(num)
		require.True(t, ok)
		require.Equal(t, num, val)
	}

	time.Sleep(ttl * 2)

	// check all entries are evicted
	for num := 1; num <= numberOfEntries; num++ {
		_, ok := cache.Get(num)
		require.False(t, ok)
	}
}

func TestEvictionCallback(t *testing.T) {
	cache := New[int, int](10, noEvictionTTL)
	evicted := make(map[int]int)

	cache.SetOnEvict(func(key int, value int) {
		evicted[key] = value
	})

	// add objects to the cache
	for i := 1; i <= 10; i++ {
		cache.Set(i, i)
	}

	// add another object to the cache
	cache.Set(11, 11)

	// check the first object is evicted
	_, ok := cache.Get(1)
	require.False(t, ok)
	require.Equal(t, 1, evicted[1])

	cache.Close()
}

func TestEvictionCallbackWithTTL(t *testing.T) {
	var mu sync.Mutex
	cache := New[int, int](10, time.Second)
	evicted := make(map[int]int)
	cache.SetOnEvict(func(key int, value int) {
		mu.Lock()
		evicted[key] = value
		mu.Unlock()
	})

	// add objects to the cache
	for i := 1; i <= 10; i++ {
		cache.Set(i, i)
	}

	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case <-timeout:
			t.Fatal("timeout")
		case <-ticker.C:
			mu.Lock()
			if len(evicted) == 10 {
				for i := 1; i <= 10; i++ {
					require.Equal(t, i, evicted[i])
				}
				return
			}
			mu.Unlock()
		}
	}
}
