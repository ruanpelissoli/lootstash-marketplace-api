package repository

import (
	"context"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/database"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
)

type ratingRepository struct {
	db *database.BunDB
}

// NewRatingRepository creates a new rating repository
func NewRatingRepository(db *database.BunDB) RatingRepository {
	return &ratingRepository{db: db}
}

func (r *ratingRepository) Create(ctx context.Context, rating *models.Rating) error {
	_, err := r.db.DB().NewInsert().
		Model(rating).
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to create rating",
			"error", err.Error(),
			"transaction_id", rating.TransactionID,
			"rater_id", rating.RaterID,
		)
	}
	return err
}

func (r *ratingRepository) GetByTransactionID(ctx context.Context, transactionID string) ([]*models.Rating, error) {
	var ratings []*models.Rating
	err := r.db.DB().NewSelect().
		Model(&ratings).
		Relation("Rater").
		Where("r.transaction_id = ?", transactionID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return ratings, nil
}

func (r *ratingRepository) GetByUserID(ctx context.Context, userID string, offset, limit int) ([]*models.Rating, int, error) {
	var ratings []*models.Rating

	query := r.db.DB().NewSelect().
		Model(&ratings).
		Relation("Rater").
		Relation("Transaction").
		Where("r.rated_id = ?", userID)

	count, err := query.Count(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to count ratings",
			"error", err.Error(),
			"rated_id", userID,
		)
		return nil, 0, err
	}

	query = query.Order("r.created_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	err = query.Scan(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to get ratings",
			"error", err.Error(),
			"rated_id", userID,
		)
		return nil, 0, err
	}

	return ratings, count, nil
}

func (r *ratingRepository) Exists(ctx context.Context, transactionID, raterID string) (bool, error) {
	exists, err := r.db.DB().NewSelect().
		Model((*models.Rating)(nil)).
		Where("transaction_id = ?", transactionID).
		Where("rater_id = ?", raterID).
		Exists(ctx)
	return exists, err
}
