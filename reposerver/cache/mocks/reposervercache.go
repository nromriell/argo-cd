package mocks

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	cacheutil "github.com/argoproj/argo-cd/v2/util/cache"
	cacheutilmocks "github.com/argoproj/argo-cd/v2/util/cache/mocks"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
)

type MockCacheType int

const (
	MockCacheTypeRedis MockCacheType = iota
	MockCacheTypeInMem
)

type MockRepoCache struct {
	mock.Mock
	RedisClient       *cacheutilmocks.MockCacheClient
	TwoLevelClient    *cacheutilmocks.MockCacheClient
	StopRedisCallback func()
}

type MockCacheOptions struct {
	RepoCacheExpiration     time.Duration
	RevisionCacheExpiration time.Duration
	ReadDelay               time.Duration
	WriteDelay              time.Duration
}

type CacheCallCounts struct {
	ExternalSets    int
	ExternalGets    int
	ExternalDeletes int
	InMemorySets    int
	InMemoryGets    int
	InMemoryDeletes int
}

// Checks that the cache was called the expected number of times
func (mockCache *MockRepoCache) AssertCacheCalledTimes(t *testing.T, calls *CacheCallCounts) {
	totalSets := calls.ExternalSets + calls.InMemorySets
	totalGets := calls.ExternalGets + calls.InMemoryGets
	totalDeletes := calls.ExternalDeletes + calls.InMemoryDeletes
	mockCache.TwoLevelClient.AssertNumberOfCalls(t, "Get", totalGets)
	mockCache.TwoLevelClient.AssertNumberOfCalls(t, "Set", totalSets)
	mockCache.TwoLevelClient.AssertNumberOfCalls(t, "Delete", totalDeletes)
	mockCache.RedisClient.AssertNumberOfCalls(t, "Get", calls.ExternalGets)
	mockCache.RedisClient.AssertNumberOfCalls(t, "Set", calls.ExternalSets)
	mockCache.RedisClient.AssertNumberOfCalls(t, "Delete", calls.ExternalDeletes)
}

func (mockCache *MockRepoCache) ConfigureDefaultCallbacks() {
	mockCache.TwoLevelClient.On("Get", mock.Anything).Return(nil)
	mockCache.TwoLevelClient.On("Set", mock.Anything).Return(nil)
	mockCache.TwoLevelClient.On("Delete", mock.Anything).Return(nil)
	mockCache.RedisClient.On("Get", mock.Anything).Return(nil)
	mockCache.RedisClient.On("Set", mock.Anything).Return(nil)
	mockCache.RedisClient.On("Delete", mock.Anything).Return(nil)
}

func NewInMemoryRedis() (*redis.Client, func()) {
	cacheutil.NewInMemoryCache(5 * time.Second)
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	return redis.NewClient(&redis.Options{Addr: mr.Addr()}), mr.Close
}

func NewMockRepoCache(cacheOpts *MockCacheOptions) *MockRepoCache {
	redisClient, stopRedis := NewInMemoryRedis()
	redisCacheClient := &cacheutilmocks.MockCacheClient{
		ReadDelay:  cacheOpts.ReadDelay,
		WriteDelay: cacheOpts.WriteDelay,
		BaseCache:  cacheutil.NewRedisCache(redisClient, cacheOpts.RepoCacheExpiration, cacheutil.RedisCompressionNone)}
	twoLevelClient := &cacheutilmocks.MockCacheClient{
		ReadDelay:  cacheOpts.ReadDelay,
		WriteDelay: cacheOpts.WriteDelay,
		BaseCache:  cacheutil.NewTwoLevelClient(redisCacheClient, cacheOpts.RepoCacheExpiration)}
	newMockCache := &MockRepoCache{TwoLevelClient: twoLevelClient, RedisClient: redisCacheClient, StopRedisCallback: stopRedis}
	newMockCache.ConfigureDefaultCallbacks()
	return newMockCache
}
