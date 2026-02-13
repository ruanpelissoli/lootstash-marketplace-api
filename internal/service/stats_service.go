package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/cache"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
)

const homeStatsTTL = 5 * time.Minute

// StatsService handles marketplace statistics business logic
type StatsService struct {
	repo        repository.StatsRepository
	redis       *cache.RedisClient
	invalidator *cache.Invalidator
}

// NewStatsService creates a new stats service
func NewStatsService(repo repository.StatsRepository, redis *cache.RedisClient) *StatsService {
	return &StatsService{
		repo:        repo,
		redis:       redis,
		invalidator: cache.NewInvalidator(redis),
	}
}

// GetMarketplaceStats retrieves marketplace statistics, checking home:stats cache first
func (s *StatsService) GetMarketplaceStats(ctx context.Context) (*dto.MarketplaceStatsResponse, error) {
	// Try cache first
	cached, err := s.redis.Get(ctx, cache.HomeStatsKey())
	if err == nil && cached != "" {
		var stats dto.MarketplaceStatsResponse
		if json.Unmarshal([]byte(cached), &stats) == nil {
			return &stats, nil
		}
	}

	// Fetch from DB
	repoStats, err := s.repo.GetMarketplaceStats(ctx)
	if err != nil {
		return nil, err
	}

	stats := &dto.MarketplaceStatsResponse{
		ActiveListings:         repoStats.ActiveListings,
		TradesToday:            repoStats.TradesToday,
		OnlineSellers:          repoStats.OnlineSellers,
		AvgResponseTimeMinutes: repoStats.AvgResponseTimeMinutes,
		LastUpdated:            time.Now(),
	}

	// Cache the result
	if data, err := json.Marshal(stats); err == nil {
		_ = s.redis.Set(ctx, cache.HomeStatsKey(), string(data), homeStatsTTL)
	}

	return stats, nil
}

// RefreshHomeStats fetches stats from DB and updates the home:stats cache
func (s *StatsService) RefreshHomeStats(ctx context.Context) {
	repoStats, err := s.repo.GetMarketplaceStats(ctx)
	if err != nil {
		logger.Log.Warn("failed to refresh home stats", "error", err.Error())
		return
	}

	stats := &dto.MarketplaceStatsResponse{
		ActiveListings:         repoStats.ActiveListings,
		TradesToday:            repoStats.TradesToday,
		AvgResponseTimeMinutes: repoStats.AvgResponseTimeMinutes,
		LastUpdated:            time.Now(),
	}

	if data, err := json.Marshal(stats); err == nil {
		_ = s.redis.Set(ctx, cache.HomeStatsKey(), string(data), homeStatsTTL)
	}
}

// WarmHomeStats populates the home:stats cache on startup
func (s *StatsService) WarmHomeStats(ctx context.Context) {
	s.RefreshHomeStats(ctx)
}
