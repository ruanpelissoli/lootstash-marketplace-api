package service

import (
	"context"
	"time"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
)

// StatsService handles marketplace statistics business logic
type StatsService struct {
	repo repository.StatsRepository
}

// NewStatsService creates a new stats service
func NewStatsService(repo repository.StatsRepository) *StatsService {
	return &StatsService{
		repo: repo,
	}
}

// GetMarketplaceStats retrieves marketplace statistics directly from pre-aggregated table
func (s *StatsService) GetMarketplaceStats(ctx context.Context) (*dto.MarketplaceStatsResponse, error) {
	repoStats, err := s.repo.GetMarketplaceStats(ctx)
	if err != nil {
		return nil, err
	}

	return &dto.MarketplaceStatsResponse{
		ActiveListings:         repoStats.ActiveListings,
		TradesToday:            repoStats.TradesToday,
		OnlineSellers:          repoStats.OnlineSellers,
		AvgResponseTimeMinutes: repoStats.AvgResponseTimeMinutes,
		LastUpdated:            time.Now(),
	}, nil
}
