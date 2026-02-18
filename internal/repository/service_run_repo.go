package repository

import (
	"context"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/database"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
)

type serviceRunRepository struct {
	db *database.BunDB
}

// NewServiceRunRepository creates a new service run repository
func NewServiceRunRepository(db *database.BunDB) ServiceRunRepository {
	return &serviceRunRepository{db: db}
}

func (r *serviceRunRepository) Create(ctx context.Context, serviceRun *models.ServiceRun) error {
	_, err := r.db.DB().NewInsert().
		Model(serviceRun).
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to create service run",
			"error", err.Error(),
			"service_id", serviceRun.ServiceID,
		)
	}
	return err
}

func (r *serviceRunRepository) GetByID(ctx context.Context, id string) (*models.ServiceRun, error) {
	serviceRun := new(models.ServiceRun)
	err := r.db.DB().NewSelect().
		Model(serviceRun).
		Where("sr.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return serviceRun, nil
}

func (r *serviceRunRepository) GetByIDWithRelations(ctx context.Context, id string) (*models.ServiceRun, error) {
	serviceRun := new(models.ServiceRun)
	err := r.db.DB().NewSelect().
		Model(serviceRun).
		Relation("Service").
		Relation("Offer").
		Relation("Provider").
		Relation("Client").
		Relation("Chat").
		Where("sr.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return serviceRun, nil
}

func (r *serviceRunRepository) Update(ctx context.Context, serviceRun *models.ServiceRun) error {
	_, err := r.db.DB().NewUpdate().
		Model(serviceRun).
		WherePK().
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to update service run",
			"error", err.Error(),
			"service_run_id", serviceRun.ID,
		)
	}
	return err
}

func (r *serviceRunRepository) List(ctx context.Context, filter ServiceRunFilter) ([]*models.ServiceRun, int, error) {
	var serviceRuns []*models.ServiceRun

	query := r.db.DB().NewSelect().
		Model(&serviceRuns).
		Relation("Service").
		Relation("Provider").
		Relation("Client").
		Relation("Offer").
		Relation("Chat")

	// Filter by role
	switch filter.Role {
	case "provider":
		query = query.Where("sr.provider_id = ?", filter.UserID)
	case "client":
		query = query.Where("sr.client_id = ?", filter.UserID)
	default:
		// All service runs where user is either provider or client
		query = query.Where("sr.provider_id = ? OR sr.client_id = ?", filter.UserID, filter.UserID)
	}

	if filter.Status != "" {
		query = query.Where("sr.status = ?", filter.Status)
	}

	count, err := query.Count(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to count service runs",
			"error", err.Error(),
			"user_id", filter.UserID,
		)
		return nil, 0, err
	}

	query = query.Order("sr.created_at DESC")

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit).Offset(filter.Offset)
	}

	err = query.Scan(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to list service runs",
			"error", err.Error(),
			"user_id", filter.UserID,
		)
		return nil, 0, err
	}

	return serviceRuns, count, nil
}
