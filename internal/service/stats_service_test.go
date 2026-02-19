package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/cache"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ensureLogger initializes the global logger if it hasn't been set.
// Required for tests that exercise error paths calling logger.Log.Warn().
func ensureLogger() {
	if logger.Log == nil {
		logger.Init("error", false)
	}
}

// ---------------------------------------------------------------------------
// GetMarketplaceStats
// ---------------------------------------------------------------------------

func TestGetMarketplaceStats_FetchesFromDB_WritesCache(t *testing.T) {
	statsRepo := new(mocks.MockStatsRepository)
	rc, mr := newTestRedisReal(t)
	svc := NewStatsService(statsRepo, rc)

	repoStats := &repository.MarketplaceStats{
		ActiveListings:         100,
		TradesToday:            25,
		OnlineSellers:          10,
		AvgResponseTimeMinutes: 3.5,
	}
	statsRepo.On("GetMarketplaceStats", mock.Anything).Return(repoStats, nil)

	result, err := svc.GetMarketplaceStats(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 100, result.ActiveListings)
	assert.Equal(t, 25, result.TradesToday)
	assert.Equal(t, 10, result.OnlineSellers)
	assert.InDelta(t, 3.5, result.AvgResponseTimeMinutes, 0.001)
	assert.False(t, result.LastUpdated.IsZero(), "LastUpdated should be set")

	// Verify cache was written
	cached, err := mr.Get("home:stats")
	assert.NoError(t, err)
	assert.NotEmpty(t, cached)

	// Unmarshal and verify cached data matches
	var cachedStats dto.MarketplaceStatsResponse
	err = json.Unmarshal([]byte(cached), &cachedStats)
	assert.NoError(t, err)
	assert.Equal(t, 100, cachedStats.ActiveListings)
	assert.Equal(t, 25, cachedStats.TradesToday)
	assert.Equal(t, 10, cachedStats.OnlineSellers)
	assert.InDelta(t, 3.5, cachedStats.AvgResponseTimeMinutes, 0.001)

	// Verify TTL was set (should be > 0 since homeStatsTTL = 5 minutes)
	ttl := mr.TTL("home:stats")
	assert.True(t, ttl > 0, "TTL should be positive")

	statsRepo.AssertExpectations(t)
}

func TestGetMarketplaceStats_ReturnsFromCache(t *testing.T) {
	statsRepo := new(mocks.MockStatsRepository)
	rc, mr := newTestRedisReal(t)
	svc := NewStatsService(statsRepo, rc)

	// Pre-populate cache with known data
	cachedData := dto.MarketplaceStatsResponse{
		ActiveListings:         200,
		TradesToday:            50,
		OnlineSellers:          20,
		AvgResponseTimeMinutes: 2.0,
	}
	data, err := json.Marshal(cachedData)
	assert.NoError(t, err)

	cacheKey := cache.HomeStatsKey()
	mr.Set(cacheKey, string(data))

	result, err := svc.GetMarketplaceStats(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 200, result.ActiveListings)
	assert.Equal(t, 50, result.TradesToday)
	assert.Equal(t, 20, result.OnlineSellers)
	assert.InDelta(t, 2.0, result.AvgResponseTimeMinutes, 0.001)

	// Repo should NOT have been called because cache was hit
	statsRepo.AssertNotCalled(t, "GetMarketplaceStats", mock.Anything)
}

func TestGetMarketplaceStats_DBError(t *testing.T) {
	statsRepo := new(mocks.MockStatsRepository)
	rc, _ := newTestRedisReal(t)
	svc := NewStatsService(statsRepo, rc)

	statsRepo.On("GetMarketplaceStats", mock.Anything).
		Return(nil, assert.AnError)

	result, err := svc.GetMarketplaceStats(context.Background())
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, assert.AnError, err)

	statsRepo.AssertExpectations(t)
}

func TestGetMarketplaceStats_CacheCorrupted_FallsBackToDB(t *testing.T) {
	statsRepo := new(mocks.MockStatsRepository)
	rc, mr := newTestRedisReal(t)
	svc := NewStatsService(statsRepo, rc)

	// Pre-populate cache with invalid JSON
	cacheKey := cache.HomeStatsKey()
	mr.Set(cacheKey, "not-valid-json")

	repoStats := &repository.MarketplaceStats{
		ActiveListings:         75,
		TradesToday:            15,
		OnlineSellers:          5,
		AvgResponseTimeMinutes: 4.0,
	}
	statsRepo.On("GetMarketplaceStats", mock.Anything).Return(repoStats, nil)

	result, err := svc.GetMarketplaceStats(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 75, result.ActiveListings)
	assert.Equal(t, 15, result.TradesToday)
	assert.Equal(t, 5, result.OnlineSellers)
	assert.InDelta(t, 4.0, result.AvgResponseTimeMinutes, 0.001)

	// Repo SHOULD have been called since cache was corrupted
	statsRepo.AssertExpectations(t)
}

func TestGetMarketplaceStats_NilRedis_FetchesFromDB(t *testing.T) {
	statsRepo := new(mocks.MockStatsRepository)
	rc := newTestRedis() // nil redis
	svc := NewStatsService(statsRepo, rc)

	repoStats := &repository.MarketplaceStats{
		ActiveListings:         50,
		TradesToday:            10,
		OnlineSellers:          3,
		AvgResponseTimeMinutes: 5.0,
	}
	statsRepo.On("GetMarketplaceStats", mock.Anything).Return(repoStats, nil)

	result, err := svc.GetMarketplaceStats(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 50, result.ActiveListings)
	assert.Equal(t, 10, result.TradesToday)

	statsRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// RefreshHomeStats
// ---------------------------------------------------------------------------

func TestRefreshHomeStats_WritesCache(t *testing.T) {
	ensureLogger()

	statsRepo := new(mocks.MockStatsRepository)
	rc, mr := newTestRedisReal(t)
	svc := NewStatsService(statsRepo, rc)

	repoStats := &repository.MarketplaceStats{
		ActiveListings:         300,
		TradesToday:            60,
		OnlineSellers:          30,
		AvgResponseTimeMinutes: 1.5,
	}
	statsRepo.On("GetMarketplaceStats", mock.Anything).Return(repoStats, nil)

	svc.RefreshHomeStats(context.Background())

	// Verify cache was written
	cached, err := mr.Get("home:stats")
	assert.NoError(t, err)
	assert.NotEmpty(t, cached)

	// Unmarshal and verify cached data
	var cachedStats dto.MarketplaceStatsResponse
	err = json.Unmarshal([]byte(cached), &cachedStats)
	assert.NoError(t, err)
	assert.Equal(t, 300, cachedStats.ActiveListings)
	assert.Equal(t, 60, cachedStats.TradesToday)
	assert.InDelta(t, 1.5, cachedStats.AvgResponseTimeMinutes, 0.001)
	assert.False(t, cachedStats.LastUpdated.IsZero(), "LastUpdated should be set")

	// Note: RefreshHomeStats does NOT set OnlineSellers in the response
	// (it is intentionally omitted in the source code)
	assert.Equal(t, 0, cachedStats.OnlineSellers)

	// Verify TTL
	ttl := mr.TTL("home:stats")
	assert.True(t, ttl > 0, "TTL should be positive")

	statsRepo.AssertExpectations(t)
}

func TestRefreshHomeStats_DBError_NoCache(t *testing.T) {
	ensureLogger()

	statsRepo := new(mocks.MockStatsRepository)
	rc, mr := newTestRedisReal(t)
	svc := NewStatsService(statsRepo, rc)

	statsRepo.On("GetMarketplaceStats", mock.Anything).
		Return(nil, assert.AnError)

	svc.RefreshHomeStats(context.Background())

	// Verify cache was NOT written
	exists := mr.Exists("home:stats")
	assert.False(t, exists, "home:stats key should not exist when DB errors")

	statsRepo.AssertExpectations(t)
}

func TestRefreshHomeStats_OverwritesExistingCache(t *testing.T) {
	ensureLogger()

	statsRepo := new(mocks.MockStatsRepository)
	rc, mr := newTestRedisReal(t)
	svc := NewStatsService(statsRepo, rc)

	// Pre-populate cache with old data
	oldData := dto.MarketplaceStatsResponse{
		ActiveListings: 10,
		TradesToday:    2,
	}
	data, _ := json.Marshal(oldData)
	mr.Set("home:stats", string(data))

	// Set up repo to return new data
	repoStats := &repository.MarketplaceStats{
		ActiveListings:         500,
		TradesToday:            100,
		OnlineSellers:          40,
		AvgResponseTimeMinutes: 0.5,
	}
	statsRepo.On("GetMarketplaceStats", mock.Anything).Return(repoStats, nil)

	svc.RefreshHomeStats(context.Background())

	// Verify cache was overwritten with new data
	cached, err := mr.Get("home:stats")
	assert.NoError(t, err)

	var cachedStats dto.MarketplaceStatsResponse
	err = json.Unmarshal([]byte(cached), &cachedStats)
	assert.NoError(t, err)
	assert.Equal(t, 500, cachedStats.ActiveListings)
	assert.Equal(t, 100, cachedStats.TradesToday)

	statsRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// NewStatsService
// ---------------------------------------------------------------------------

func TestNewStatsService_ReturnsValidInstance(t *testing.T) {
	statsRepo := new(mocks.MockStatsRepository)
	rc, _ := newTestRedisReal(t)

	svc := NewStatsService(statsRepo, rc)
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.repo)
	assert.NotNil(t, svc.redis)
	assert.NotNil(t, svc.invalidator)
}
