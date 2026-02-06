package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// BunDB wraps the bun.DB connection
type BunDB struct {
	db *bun.DB
}

// NewBunDB creates a new Bun database connection
func NewBunDB(ctx context.Context, databaseURL string, enableQueryLogging bool) (*BunDB, error) {
	connector := pgdriver.NewConnector(
		pgdriver.WithDSN(databaseURL),
		pgdriver.WithInsecure(true),
	)
	sqldb := sql.OpenDB(connector)

	// Configure connection pool
	sqldb.SetMaxOpenConns(25)
	sqldb.SetMaxIdleConns(5)

	db := bun.NewDB(sqldb, pgdialect.New())

	// Add query logging hook if debug logging is enabled
	if enableQueryLogging {
		db.AddQueryHook(&queryLoggingHook{})
	}

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &BunDB{db: db}, nil
}

// queryLoggingHook logs database queries for debugging
type queryLoggingHook struct{}

func (h *queryLoggingHook) BeforeQuery(ctx context.Context, event *bun.QueryEvent) context.Context {
	return ctx
}

func (h *queryLoggingHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	duration := time.Since(event.StartTime)
	log := logger.FromContext(ctx)

	if event.Err != nil {
		log.Error("database query failed",
			"query", event.Query,
			"duration_ms", duration.Milliseconds(),
			"error", event.Err.Error(),
		)
	} else {
		log.Debug("database query executed",
			"query", event.Query,
			"duration_ms", duration.Milliseconds(),
		)
	}
}

// DB returns the underlying bun.DB
func (b *BunDB) DB() *bun.DB {
	return b.db
}

// Close closes the database connection
func (b *BunDB) Close() error {
	return b.db.Close()
}

// BeginTx starts a new transaction
func (b *BunDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (bun.Tx, error) {
	return b.db.BeginTx(ctx, opts)
}
