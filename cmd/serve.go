package cmd

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/cache"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/database"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/storage"
	"github.com/spf13/cobra"
)

var (
	port           int
	allowedOrigins string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP API server",
	Long: `Start the LootStash Marketplace API HTTP server.

The server exposes REST endpoints for trading operations.

Public Endpoints:
  GET /health                          - Health check
  GET /api/v1/listings                 - List/filter listings
  GET /api/v1/listings/:id             - Get listing details
  GET /api/v1/profiles/:id             - Get user profile
  GET /api/v1/profiles/:id/ratings     - Get user ratings
  GET /api/v1/decline-reasons          - Get decline reason list
  GET /api/v1/games/:game/categories   - Get game categories

Authenticated Endpoints:
  GET    /api/v1/me                    - Get current user profile
  PATCH  /api/v1/me                    - Update current user profile
  POST   /api/v1/listings              - Create listing
  PATCH  /api/v1/listings/:id          - Update listing
  DELETE /api/v1/listings/:id          - Cancel listing
  GET    /api/v1/my/listings           - Get my listings
  POST   /api/v1/trades                - Create trade request
  GET    /api/v1/trades                - List my trade requests
  GET    /api/v1/trades/:id            - Get trade details
  POST   /api/v1/trades/:id/accept     - Accept offer
  POST   /api/v1/trades/:id/reject     - Reject offer
  POST   /api/v1/trades/:id/complete   - Mark trade complete
  POST   /api/v1/messages              - Send message
  GET    /api/v1/messages/trade/:id    - Get trade messages
  POST   /api/v1/messages/read         - Mark messages read
  GET    /api/v1/notifications         - List notifications
  GET    /api/v1/notifications/count   - Unread count
  POST   /api/v1/notifications/read    - Mark as read
  POST   /api/v1/ratings               - Rate a completed trade

Examples:
  # Start with default settings (port 8081)
  lootstash-marketplace serve

  # Start on custom port
  lootstash-marketplace serve --port 3003`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().IntVar(&port, "port", 8081, "Port to listen on")
	serveCmd.Flags().StringVar(&allowedOrigins, "allowed-origins", "*", "Comma-separated list of allowed CORS origins")
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Initialize logger
	logger.Init(GetLogLevel(), GetLogJSON())
	log := logger.Log

	log.Info("starting lootstash marketplace api",
		"version", "1.0.0",
		"log_level", GetLogLevel(),
		"log_json", GetLogJSON(),
	)

	// Connect to database
	log.Info("connecting to database")
	db, err := database.NewBunDB(ctx, GetDatabaseURL(), logger.IsDebugEnabled())
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		return err
	}
	defer func() {
		log.Info("closing database connection")
		db.Close()
	}()
	log.Info("connected to database")

	// Connect to Redis (optional - app works without it)
	log.Info("connecting to redis")
	redisClient, err := cache.NewRedisClient(ctx, GetRedisURL())
	if err != nil {
		log.Warn("redis unavailable, running without cache", "error", err)
		redisClient = nil
	} else {
		defer func() {
			log.Info("closing redis connection")
			redisClient.Close()
		}()
		log.Info("connected to redis", "address", GetRedisURL())
	}

	// Initialize storage for avatars using S3 protocol
	supabaseURL := GetSupabaseURL()
	s3AccessKey := os.Getenv("SUPABASE_S3_ACCESS_KEY")
	s3SecretKey := os.Getenv("SUPABASE_S3_SECRET_KEY")
	var avatarStorage storage.Storage
	if s3AccessKey != "" && s3SecretKey != "" {
		s3Endpoint := supabaseURL + "/storage/v1/s3"
		s3Region := os.Getenv("SUPABASE_S3_REGION")
		if s3Region == "" {
			s3Region = "local"
		}
		var err error
		avatarStorage, err = storage.NewS3Storage(s3Endpoint, s3AccessKey, s3SecretKey, s3Region, "avatars", supabaseURL)
		if err != nil {
			log.Error("failed to initialize S3 storage", "error", err)
		} else {
			log.Info("avatar storage initialized (S3)", "bucket", "avatars")
		}
	} else {
		log.Warn("SUPABASE_S3_ACCESS_KEY or SUPABASE_S3_SECRET_KEY not set, avatar uploads will be disabled")
	}

	// Create server config
	authDebug := strings.ToLower(os.Getenv("AUTH_DEBUG")) == "true"
	config := &api.Config{
		Port:                  port,
		AllowedOrigins:        allowedOrigins,
		JWTSecret:             GetJWTSecret(),
		JWKSURL:               supabaseURL + "/auth/v1/.well-known/jwks.json",
		JWTAudience:           "authenticated",
		JWTIssuer:             supabaseURL + "/auth/v1",
		AuthDebug:             authDebug,
		SupabaseURL:           supabaseURL,
		BattleNetClientID:     GetBattleNetClientID(),
		BattleNetClientSecret: GetBattleNetClientSecret(),
		BattleNetRedirectURI:  GetBattleNetRedirectURI(),
		StripeSecretKey:       GetStripeSecretKey(),
		StripeWebhookSecret:   GetStripeWebhookSecret(),
		StripePriceID:         GetStripePriceID(),
		StripeSuccessURL:      GetStripeSuccessURL(),
		StripeCancelURL:       GetStripeCancelURL(),
	}

	// Create and start server
	server := api.NewServer(db, redisClient, avatarStorage, config)

	// Handle graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-shutdown
		log.Info("received shutdown signal", "signal", sig.String())
		log.Info("shutting down server gracefully")
		if err := server.Shutdown(); err != nil {
			log.Error("error during shutdown", "error", err)
		}
		log.Info("server shutdown complete")
	}()

	log.Info("starting http server",
		"port", port,
		"allowed_origins", allowedOrigins,
	)

	if err := server.Start(); err != nil {
		log.Error("server error", "error", err)
		return err
	}

	return nil
}
