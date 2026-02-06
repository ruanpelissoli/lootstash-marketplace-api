package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	databaseURL           string
	redisURL              string
	supabaseURL           string
	jwtSecret             string
	logLevel              string
	logJSON               bool
	battleNetClientID     string
	battleNetClientSecret string
	battleNetRedirectURI  string
	stripeSecretKey       string
	stripeWebhookSecret   string
	stripePriceID         string
	stripeSuccessURL      string
	stripeCancelURL       string
)

var rootCmd = &cobra.Command{
	Use:   "lootstash-marketplace",
	Short: "LootStash Marketplace API - Trading platform for ARPG items",
	Long: `LootStash Marketplace API enables peer-to-peer trading for ARPG games.
It provides endpoints for listings, trade requests, messaging, and ratings.

Supported games:
  - diablo2: Diablo II: Resurrected`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&databaseURL, "database-url", getEnvOrDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:54322/postgres"), "PostgreSQL connection string")
	rootCmd.PersistentFlags().StringVar(&redisURL, "redis-url", getEnvOrDefault("REDIS_URL", "localhost:6379"), "Redis connection string")
	rootCmd.PersistentFlags().StringVar(&supabaseURL, "supabase-url", getEnvOrDefault("SUPABASE_URL", "http://127.0.0.1:54321"), "Supabase URL")
	rootCmd.PersistentFlags().StringVar(&jwtSecret, "jwt-secret", getEnvOrDefault("SUPABASE_JWT_SECRET", ""), "JWT secret for token verification")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", getEnvOrDefault("LOG_LEVEL", "error"), "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolVar(&logJSON, "log-json", getEnvOrDefaultBool("LOG_JSON", true), "Output logs in JSON format")
	rootCmd.PersistentFlags().StringVar(&battleNetClientID, "battlenet-client-id", getEnvOrDefault("BATTLENET_CLIENT_ID", ""), "Battle.net OAuth client ID")
	rootCmd.PersistentFlags().StringVar(&battleNetClientSecret, "battlenet-client-secret", getEnvOrDefault("BATTLENET_CLIENT_SECRET", ""), "Battle.net OAuth client secret")
	rootCmd.PersistentFlags().StringVar(&battleNetRedirectURI, "battlenet-redirect-uri", getEnvOrDefault("BATTLENET_REDIRECT_URI", ""), "Battle.net OAuth redirect URI")
	rootCmd.PersistentFlags().StringVar(&stripeSecretKey, "stripe-secret-key", getEnvOrDefault("STRIPE_SECRET_KEY", ""), "Stripe secret key")
	rootCmd.PersistentFlags().StringVar(&stripeWebhookSecret, "stripe-webhook-secret", getEnvOrDefault("STRIPE_WEBHOOK_SECRET", ""), "Stripe webhook signing secret")
	rootCmd.PersistentFlags().StringVar(&stripePriceID, "stripe-price-id", getEnvOrDefault("STRIPE_PRICE_ID", ""), "Stripe price ID for premium subscription")
	rootCmd.PersistentFlags().StringVar(&stripeSuccessURL, "stripe-success-url", getEnvOrDefault("STRIPE_SUCCESS_URL", ""), "URL to redirect after successful checkout")
	rootCmd.PersistentFlags().StringVar(&stripeCancelURL, "stripe-cancel-url", getEnvOrDefault("STRIPE_CANCEL_URL", ""), "URL to redirect after cancelled checkout")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvOrDefaultBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1"
}

func GetDatabaseURL() string {
	return databaseURL
}

func GetRedisURL() string {
	return redisURL
}

func GetSupabaseURL() string {
	return supabaseURL
}

func GetJWTSecret() string {
	return jwtSecret
}

func GetLogLevel() string {
	return logLevel
}

func GetLogJSON() bool {
	return logJSON
}

func GetBattleNetClientID() string {
	if battleNetClientID == "" {
		return os.Getenv("BATTLENET_CLIENT_ID")
	}
	return battleNetClientID
}

func GetBattleNetClientSecret() string {
	if battleNetClientSecret == "" {
		return os.Getenv("BATTLENET_CLIENT_SECRET")
	}
	return battleNetClientSecret
}

func GetBattleNetRedirectURI() string {
	if battleNetRedirectURI == "" {
		return os.Getenv("BATTLENET_REDIRECT_URI")
	}
	return battleNetRedirectURI
}

func GetStripeSecretKey() string {
	if stripeSecretKey == "" {
		return os.Getenv("STRIPE_SECRET_KEY")
	}
	return stripeSecretKey
}

func GetStripeWebhookSecret() string {
	if stripeWebhookSecret == "" {
		return os.Getenv("STRIPE_WEBHOOK_SECRET")
	}
	return stripeWebhookSecret
}

func GetStripePriceID() string {
	if stripePriceID == "" {
		return os.Getenv("STRIPE_PRICE_ID")
	}
	return stripePriceID
}

func GetStripeSuccessURL() string {
	if stripeSuccessURL == "" {
		return os.Getenv("STRIPE_SUCCESS_URL")
	}
	return stripeSuccessURL
}

func GetStripeCancelURL() string {
	if stripeCancelURL == "" {
		return os.Getenv("STRIPE_CANCEL_URL")
	}
	return stripeCancelURL
}

func PrintSuccess(msg string) {
	fmt.Printf("✓ %s\n", msg)
}

func PrintError(msg string) {
	fmt.Printf("✗ %s\n", msg)
}

func PrintInfo(msg string) {
	fmt.Printf("→ %s\n", msg)
}
