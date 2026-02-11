package repository

import (
	"context"
	"time"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/database"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
)

type bugReportRepository struct {
	db *database.BunDB
}

// NewBugReportRepository creates a new bug report repository
func NewBugReportRepository(db *database.BunDB) BugReportRepository {
	return &bugReportRepository{db: db}
}

func (r *bugReportRepository) Create(ctx context.Context, report *models.BugReport) error {
	_, err := r.db.DB().NewInsert().
		Model(report).
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to create bug report",
			"error", err.Error(),
			"user_id", report.UserID,
		)
	}
	return err
}

func (r *bugReportRepository) GetByID(ctx context.Context, id string) (*models.BugReport, error) {
	report := new(models.BugReport)
	err := r.db.DB().NewSelect().
		Model(report).
		Relation("Reporter").
		Where("br.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return report, nil
}

func (r *bugReportRepository) Update(ctx context.Context, report *models.BugReport) error {
	report.UpdatedAt = time.Now()
	_, err := r.db.DB().NewUpdate().
		Model(report).
		WherePK().
		Exec(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to update bug report",
			"error", err.Error(),
			"bug_report_id", report.ID,
		)
	}
	return err
}

func (r *bugReportRepository) List(ctx context.Context, status string, offset, limit int) ([]*models.BugReport, int, error) {
	var reports []*models.BugReport

	q := r.db.DB().NewSelect().
		Model(&reports).
		Relation("Reporter").
		OrderExpr("br.created_at DESC")

	if status != "" {
		q = q.Where("br.status = ?", status)
	}

	count, err := q.Count(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to count bug reports",
			"error", err.Error(),
		)
		return nil, 0, err
	}

	err = q.Offset(offset).Limit(limit).Scan(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("failed to list bug reports",
			"error", err.Error(),
		)
		return nil, 0, err
	}

	return reports, count, nil
}
