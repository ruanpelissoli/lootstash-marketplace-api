package api

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/handlers/v1"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/middleware"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/cache"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/database"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/games"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/games/d2"
	applogger "github.com/ruanpelissoli/lootstash-marketplace-api/internal/logger"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/service"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/storage"
)

// Server represents the HTTP server
type Server struct {
	app     *fiber.App
	db      *database.BunDB
	redis   *cache.RedisClient
	storage storage.Storage
	config  *Config
}

// Config holds server configuration
type Config struct {
	Port           int
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	AllowedOrigins string
	JWTSecret      string // For HMAC (HS256) - legacy support
	JWKSURL        string // For ECDSA (ES256) - Supabase JWKS endpoint
	JWTAudience    string // Expected "aud" claim (e.g., "authenticated")
	JWTIssuer      string // Expected "iss" claim (optional)
	AuthDebug      bool   // Enable auth debug logging
	SupabaseURL    string // Supabase URL for storage URLs
	// Battle.net OAuth configuration
	BattleNetClientID     string
	BattleNetClientSecret string
	BattleNetRedirectURI  string
	// Stripe configuration
	StripeSecretKey    string
	StripeWebhookSecret string
	StripePriceID      string
	StripeSuccessURL   string
	StripeCancelURL    string
}

// DefaultConfig returns default server configuration
func DefaultConfig() *Config {
	return &Config{
		Port:           8081,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		AllowedOrigins: "*",
	}
}

// NewServer creates a new HTTP server
func NewServer(db *database.BunDB, redis *cache.RedisClient, stor storage.Storage, config *Config) *Server {
	if config == nil {
		config = DefaultConfig()
	}

	app := fiber.New(fiber.Config{
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		AppName:      "LootStash Marketplace API",
	})

	server := &Server{
		app:     app,
		db:      db,
		redis:   redis,
		storage: stor,
		config:  config,
	}

	server.setupMiddleware()
	server.setupRoutes()

	return server
}

func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.app.Use(recover.New())

	// Logger middleware - simple format like catalog-api
	s.app.Use(logger.New(logger.Config{
		Format:     "${time} ${status} ${method} ${path}\t${latency}\n",
		TimeFormat: "2006-01-02 15:04:05",
	}))

	// Request ID middleware
	s.app.Use(middleware.RequestID())

	// CORS middleware
	s.app.Use(middleware.NewCORSMiddleware(middleware.CORSConfig{
		AllowedOrigins: s.config.AllowedOrigins,
	}))
}

func (s *Server) setupRoutes() {
	// Register game handlers
	registry := games.GetRegistry()
	d2.Register(registry)

	// Create repositories
	profileRepo := repository.NewProfileRepository(s.db)
	listingRepo := repository.NewListingRepository(s.db)
	offerRepo := repository.NewOfferRepository(s.db)
	tradeRepo := repository.NewTradeRepositoryNew(s.db)
	chatRepo := repository.NewChatRepository(s.db)
	messageRepo := repository.NewMessageRepository(s.db)
	notificationRepo := repository.NewNotificationRepository(s.db)
	transactionRepo := repository.NewTransactionRepository(s.db)
	ratingRepo := repository.NewRatingRepository(s.db)
	statsRepo := repository.NewStatsRepository(s.db)
	billingEventRepo := repository.NewBillingEventRepository(s.db)

	// Create repositories (wishlist)
	wishlistRepo := repository.NewWishlistRepository(s.db)

	// Create services
	profileService := service.NewProfileService(profileRepo, s.redis, s.storage)
	notificationService := service.NewNotificationService(notificationRepo, s.redis)
	listingService := service.NewListingService(listingRepo, profileService, s.redis)
	wishlistService := service.NewWishlistService(wishlistRepo, profileService, notificationService)
	listingService.SetWishlistService(wishlistService)
	statsService := service.NewStatsService(statsRepo)
	offerService := service.NewOfferService(
		s.db,
		offerRepo,
		listingRepo,
		tradeRepo,
		chatRepo,
		notificationService,
		profileService,
		listingService,
		s.redis,
	)
	tradeService := service.NewTradeServiceNew(
		s.db,
		tradeRepo,
		listingRepo,
		transactionRepo,
		ratingRepo,
		chatRepo,
		notificationService,
		profileService,
		listingService,
		s.redis,
		s.config.SupabaseURL,
	)
	chatService := service.NewChatService(chatRepo, messageRepo, tradeRepo, profileService, notificationService)
	ratingService := service.NewRatingService(ratingRepo, transactionRepo, profileService, notificationService)
	battleNetService := service.NewBattleNetService(
		service.BattleNetConfig{
			ClientID:     s.config.BattleNetClientID,
			ClientSecret: s.config.BattleNetClientSecret,
			RedirectURI:  s.config.BattleNetRedirectURI,
		},
		s.redis,
		profileRepo,
	)
	subscriptionService := service.NewSubscriptionService(
		profileRepo,
		billingEventRepo,
		transactionRepo,
		s.redis,
		service.StripeConfig{
			SecretKey:     s.config.StripeSecretKey,
			WebhookSecret: s.config.StripeWebhookSecret,
			PriceID:       s.config.StripePriceID,
			SuccessURL:    s.config.StripeSuccessURL,
			CancelURL:     s.config.StripeCancelURL,
		},
	)

	// Create handlers
	profileHandler := v1.NewProfileHandler(profileService)
	listingHandler := v1.NewListingHandler(listingService)
	offerHandler := v1.NewOfferHandler(offerService)
	tradeHandler := v1.NewTradeHandlerNew(tradeService)
	chatHandler := v1.NewChatHandler(chatService)
	notificationHandler := v1.NewNotificationHandler(notificationService)
	ratingHandler := v1.NewRatingHandler(ratingService)
	statsHandler := v1.NewStatsHandler(statsService)
	battleNetHandler := v1.NewBattleNetHandler(battleNetService)
	subscriptionHandler := v1.NewSubscriptionHandler(subscriptionService)
	webhookHandler := v1.NewWebhookHandler(subscriptionService)
	wishlistHandler := v1.NewWishlistHandler(wishlistService)
	premiumHandler := v1.NewPremiumHandler(subscriptionService, listingService)

	// Auth middleware config
	authConfig := middleware.AuthConfig{
		JWTSecret: s.config.JWTSecret,
		JWKSURL:   s.config.JWKSURL,
		Audience:  s.config.JWTAudience,
		Issuer:    s.config.JWTIssuer,
		Debug:     s.config.AuthDebug,
	}
	authRequired := middleware.NewAuthMiddleware(authConfig)
	authOptional := middleware.OptionalAuthMiddleware(authConfig)

	// Activity tracking middleware (updates last_active_at for online sellers count)
	activityTracker := middleware.ActivityTracker(profileRepo, s.redis)

	// Health check
	s.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "lootstash-marketplace-api",
		})
	})

	// API v1 routes
	api := s.app.Group("/api")
	apiV1 := api.Group("/v1")

	// Public routes
	apiV1.Post("/listings/search", authOptional, listingHandler.Search)
	apiV1.Get("/listings", authOptional, listingHandler.List)
	apiV1.Get("/listings/:id", authOptional, listingHandler.GetByID)
	apiV1.Get("/profiles/:id", profileHandler.GetByID)
	apiV1.Get("/profiles/:id/ratings", ratingHandler.GetByProfileID)
	apiV1.Get("/decline-reasons", offerHandler.GetDeclineReasons)
	apiV1.Get("/marketplace/stats", statsHandler.GetMarketplaceStats)

	// Stripe webhook (no auth required)
	apiV1.Post("/webhooks/stripe", webhookHandler.StripeWebhook)

	// Game routes
	apiV1.Get("/games/:game/categories", func(c *fiber.Ctx) error {
		gameCode := c.Params("game")
		categories, err := registry.GetCategories(gameCode)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "not_found",
				"message": "Game not found",
				"code":    404,
			})
		}
		return c.JSON(categories)
	})

	// Authenticated routes (with activity tracking for online sellers count)
	authenticated := apiV1.Group("", authRequired, activityTracker)

	// Profile routes
	authenticated.Get("/me", profileHandler.GetMe)
	authenticated.Patch("/me", profileHandler.UpdateMe)
	authenticated.Post("/me/picture", profileHandler.UploadPicture)

	// Battle.net OAuth routes
	authenticated.Post("/me/battlenet/link", battleNetHandler.Link)
	authenticated.Post("/me/battlenet/callback", battleNetHandler.Callback)
	authenticated.Delete("/me/battlenet", battleNetHandler.Unlink)

	// My listings
	authenticated.Get("/my/listings", listingHandler.ListMy)

	// Listing management
	authenticated.Post("/listings", listingHandler.Create)
	authenticated.Patch("/listings/:id", listingHandler.Update)
	authenticated.Delete("/listings/:id", listingHandler.Delete)

	// Offer routes
	authenticated.Get("/offers", offerHandler.List)
	authenticated.Post("/offers", offerHandler.Create)
	authenticated.Get("/offers/:id", offerHandler.GetByID)
	authenticated.Post("/offers/:id/accept", offerHandler.Accept)
	authenticated.Post("/offers/:id/reject", offerHandler.Reject)
	authenticated.Post("/offers/:id/cancel", offerHandler.Cancel)

	// Trade routes
	authenticated.Get("/trades", tradeHandler.List)
	authenticated.Get("/trades/:id", tradeHandler.GetByID)
	authenticated.Post("/trades/:id/complete", tradeHandler.Complete)
	authenticated.Post("/trades/:id/cancel", tradeHandler.Cancel)

	// Chat routes
	authenticated.Get("/chats/:id", chatHandler.GetByID)
	authenticated.Get("/chats/:id/messages", chatHandler.GetMessages)
	authenticated.Post("/chats/:id/messages", chatHandler.SendMessage)
	authenticated.Post("/chats/:id/read", chatHandler.MarkRead)

	// Notification routes
	authenticated.Get("/notifications", notificationHandler.List)
	authenticated.Get("/notifications/count", notificationHandler.Count)
	authenticated.Post("/notifications/read", notificationHandler.MarkRead)

	// Rating routes
	authenticated.Post("/ratings", ratingHandler.Create)

	// Subscription routes
	authenticated.Get("/subscriptions/me", subscriptionHandler.GetMe)
	authenticated.Post("/subscriptions/checkout", subscriptionHandler.Checkout)
	authenticated.Post("/subscriptions/cancel", subscriptionHandler.Cancel)
	authenticated.Get("/subscriptions/billing-history", subscriptionHandler.BillingHistory)

	// Wishlist routes
	authenticated.Get("/wishlist", wishlistHandler.List)
	authenticated.Post("/wishlist", wishlistHandler.Create)
	authenticated.Patch("/wishlist/:id", wishlistHandler.Update)
	authenticated.Delete("/wishlist/:id", wishlistHandler.Delete)

	// Premium feature routes
	authenticated.Patch("/me/flair", premiumHandler.UpdateFlair)
	authenticated.Get("/marketplace/price-history", premiumHandler.PriceHistory)
	authenticated.Get("/my/listings/count", premiumHandler.ListingCount)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Port)
	applogger.Log.Info("http server listening", "address", addr)
	return s.app.Listen(addr)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	return s.app.Shutdown()
}
