package repository

import (
	"context"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/database"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

type serviceRepository struct {
	db *database.BunDB
}

// NewServiceRepository creates a new service repository
func NewServiceRepository(db *database.BunDB) ServiceRepository {
	return &serviceRepository{db: db}
}

func (r *serviceRepository) Create(ctx context.Context, service *models.Service) error {
	_, err := r.db.DB().NewInsert().
		Model(service).
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to create service",
			"error", err.Error(),
			"provider_id", service.ProviderID,
		)
	}
	return err
}

func (r *serviceRepository) GetByID(ctx context.Context, id string) (*models.Service, error) {
	service := new(models.Service)
	err := r.db.DB().NewSelect().
		Model(service).
		Where("s.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return service, nil
}

func (r *serviceRepository) GetByIDWithProvider(ctx context.Context, id string) (*models.Service, error) {
	service := new(models.Service)
	err := r.db.DB().NewSelect().
		Model(service).
		Relation("Provider").
		Where("s.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return service, nil
}

func (r *serviceRepository) Update(ctx context.Context, service *models.Service) error {
	_, err := r.db.DB().NewUpdate().
		Model(service).
		WherePK().
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to update service",
			"error", err.Error(),
			"service_id", service.ID,
		)
	}
	return err
}

func (r *serviceRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.DB().NewDelete().
		Model((*models.Service)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

func (r *serviceRepository) ListByProviderID(ctx context.Context, providerID string, offset, limit int) ([]*models.Service, int, error) {
	var services []*models.Service

	query := r.db.DB().NewSelect().
		Model(&services).
		Where("s.provider_id = ?", providerID).
		Where("s.status != ?", "cancelled")

	count, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	query = query.Order("s.created_at DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	err = query.Scan(ctx)
	if err != nil {
		return nil, 0, err
	}

	return services, count, nil
}

func (r *serviceRepository) ListProviders(ctx context.Context, filter ServiceProviderFilter) ([]ProviderWithServices, int, error) {
	// Step 1: Get distinct provider IDs matching filters, sorted by premium + rating, paginated
	subQuery := r.db.DB().NewSelect().
		ColumnExpr("DISTINCT s.provider_id").
		TableExpr("d2.services AS s").
		Join("JOIN d2.profiles AS p ON p.id = s.provider_id").
		Where("s.status = ?", "active")

	// Apply filters
	if len(filter.ServiceType) > 0 {
		subQuery = subQuery.Where("s.service_type IN (?)", bun.In(filter.ServiceType))
	}

	game := filter.Game
	if game == "" {
		game = "diablo2"
	}
	subQuery = subQuery.Where("s.game = ?", game)

	if filter.Ladder != nil {
		subQuery = subQuery.Where("s.ladder = ?", *filter.Ladder)
	}
	if filter.Hardcore != nil {
		subQuery = subQuery.Where("s.hardcore = ?", *filter.Hardcore)
	}
	if filter.IsNonRotw != nil {
		subQuery = subQuery.Where("s.is_non_rotw = ?", *filter.IsNonRotw)
	}
	if len(filter.Platforms) > 0 {
		subQuery = subQuery.Where("s.platforms && ?", pgdialect.Array(filter.Platforms))
	}
	if filter.Region != "" {
		subQuery = subQuery.Where("s.region = ?", filter.Region)
	}

	// Count total distinct providers
	countQuery := r.db.DB().NewSelect().
		ColumnExpr("COUNT(*)").
		TableExpr("(?) AS sub", subQuery)

	var totalCount int
	err := countQuery.Scan(ctx, &totalCount)
	if err != nil {
		logger.FromContext(ctx).Error("failed to count service providers",
			"error", err.Error(),
		)
		return nil, 0, err
	}

	// Get paginated provider IDs sorted by premium + rating
	var providerIDs []string
	err = r.db.DB().NewSelect().
		ColumnExpr("sub.provider_id").
		TableExpr("(?) AS sub", subQuery).
		Join("JOIN d2.profiles AS p ON p.id = sub.provider_id").
		OrderExpr("p.is_premium DESC, p.average_rating DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Scan(ctx, &providerIDs)
	if err != nil {
		logger.FromContext(ctx).Error("failed to list service provider IDs",
			"error", err.Error(),
		)
		return nil, 0, err
	}

	if len(providerIDs) == 0 {
		return []ProviderWithServices{}, totalCount, nil
	}

	// Step 2: Fetch profiles for these providers
	var profiles []*models.Profile
	err = r.db.DB().NewSelect().
		Model(&profiles).
		Where("id IN (?)", bun.In(providerIDs)).
		Scan(ctx)
	if err != nil {
		return nil, 0, err
	}

	profileMap := make(map[string]*models.Profile)
	for _, p := range profiles {
		profileMap[p.ID] = p
	}

	// Step 3: Fetch all active services for those providers
	var services []*models.Service
	err = r.db.DB().NewSelect().
		Model(&services).
		Where("s.provider_id IN (?)", bun.In(providerIDs)).
		Where("s.status = ?", "active").
		Order("s.created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Group services by provider
	servicesByProvider := make(map[string][]*models.Service)
	for _, s := range services {
		servicesByProvider[s.ProviderID] = append(servicesByProvider[s.ProviderID], s)
	}

	// Assemble results in the same order as providerIDs
	results := make([]ProviderWithServices, 0, len(providerIDs))
	for _, pid := range providerIDs {
		profile := profileMap[pid]
		if profile == nil {
			continue
		}
		results = append(results, ProviderWithServices{
			Provider: profile,
			Services: servicesByProvider[pid],
		})
	}

	return results, totalCount, nil
}

func (r *serviceRepository) GetProviderServices(ctx context.Context, providerID string) ([]*models.Service, error) {
	var services []*models.Service
	err := r.db.DB().NewSelect().
		Model(&services).
		Where("s.provider_id = ?", providerID).
		Where("s.status = ?", "active").
		Order("s.created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return services, nil
}

func (r *serviceRepository) ExistsByProviderAndType(ctx context.Context, providerID string, serviceType string, game string) (bool, error) {
	exists, err := r.db.DB().NewSelect().
		Model((*models.Service)(nil)).
		Where("provider_id = ?", providerID).
		Where("service_type = ?", serviceType).
		Where("game = ?", game).
		Where("status IN (?)", bun.In([]string{"active", "paused"})).
		Exists(ctx)
	return exists, err
}
