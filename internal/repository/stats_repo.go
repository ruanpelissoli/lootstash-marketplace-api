package repository

import (
	"context"
	"time"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/database"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
)

type statsRepository struct {
	db *database.BunDB
}

// NewStatsRepository creates a new stats repository
func NewStatsRepository(db *database.BunDB) StatsRepository {
	return &statsRepository{db: db}
}

func (r *statsRepository) GetMarketplaceStats(ctx context.Context) (*MarketplaceStats, error) {
	// Try to read from pre-aggregated marketplace_stats table first
	stats, err := r.getPreAggregatedStats(ctx)
	if err == nil && stats != nil {
		return stats, nil
	}

	// Log the fallback if pre-aggregated stats failed
	if err != nil {
		logger.FromContext(ctx).Warn("falling back to live queries for stats", "error", err.Error())
	}

	// Fallback to live queries
	return r.getLiveStats(ctx)
}

// getPreAggregatedStats reads from the d2.marketplace_stats table
func (r *statsRepository) getPreAggregatedStats(ctx context.Context) (*MarketplaceStats, error) {
	var dbStats models.MarketplaceStats

	err := r.db.DB().NewSelect().
		Model(&dbStats).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	// Online sellers still needs live calculation since it's based on recent activity
	onlineSellers, err := r.countOnlineSellers(ctx)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to count online sellers", "error", err.Error())
		onlineSellers = 0
	}

	return &MarketplaceStats{
		ActiveListings:         dbStats.ActiveListings,
		TradesToday:            dbStats.TradesToday,
		OnlineSellers:          onlineSellers,
		AvgResponseTimeMinutes: dbStats.AvgResponseTimeMinutes,
	}, nil
}

// getLiveStats calculates stats via direct queries (fallback)
func (r *statsRepository) getLiveStats(ctx context.Context) (*MarketplaceStats, error) {
	stats := &MarketplaceStats{}

	// Count active listings
	activeListings, err := r.db.DB().NewSelect().
		Model((*models.Listing)(nil)).
		Where("status = ?", "active").
		Count(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to count active listings", "error", err.Error())
		return nil, err
	}
	stats.ActiveListings = activeListings

	// Count completed trades today
	today := time.Now().Truncate(24 * time.Hour)
	tradesToday, err := r.db.DB().NewSelect().
		Model((*models.Trade)(nil)).
		Where("status = ?", "completed").
		Where("completed_at >= ?", today).
		Count(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to count trades today", "error", err.Error())
		return nil, err
	}
	stats.TradesToday = tradesToday

	// Count online sellers
	onlineSellers, err := r.countOnlineSellers(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to count online sellers", "error", err.Error())
		return nil, err
	}
	stats.OnlineSellers = onlineSellers

	// Calculate average response time (in minutes) for accepted offers
	// Response time = time between offer creation and acceptance
	var avgResponse struct {
		AvgMinutes float64 `bun:"avg_minutes"`
	}
	err = r.db.DB().NewSelect().
		Model((*models.Offer)(nil)).
		ColumnExpr("COALESCE(AVG(EXTRACT(EPOCH FROM (accepted_at - created_at)) / 60), 0) as avg_minutes").
		Where("status = ?", "accepted").
		Where("accepted_at IS NOT NULL").
		Where("created_at >= ?", time.Now().Add(-7*24*time.Hour)). // Last 7 days
		Scan(ctx, &avgResponse)
	if err != nil {
		logger.FromContext(ctx).Error("failed to calculate avg response time", "error", err.Error())
		return nil, err
	}
	stats.AvgResponseTimeMinutes = avgResponse.AvgMinutes

	return stats, nil
}

// countOnlineSellers counts profiles with activity in the last 15 minutes
func (r *statsRepository) countOnlineSellers(ctx context.Context) (int, error) {
	fifteenMinAgo := time.Now().Add(-15 * time.Minute)
	return r.db.DB().NewSelect().
		Model((*models.Profile)(nil)).
		Where("last_active_at >= ?", fifteenMinAgo).
		Count(ctx)
}
