package middleware

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
)

const (
	// UserIDKey is the key used to store the user ID in fiber context
	UserIDKey = "user_id"
)

// AuthConfig holds auth middleware configuration
type AuthConfig struct {
	JWTSecret string // For HMAC (HS256) - legacy support
	JWKSURL   string // For ECDSA (ES256) - Supabase JWKS endpoint
	Audience  string // Expected "aud" claim (e.g., "authenticated")
	Issuer    string // Expected "iss" claim (optional)
	Debug     bool   // Enable debug logging
}

// jwksCache holds cached JWKS keys
type jwksCache struct {
	keys      map[string]*ecdsa.PublicKey
	fetchedAt time.Time
	mu        sync.RWMutex
}

var globalJWKSCache = &jwksCache{
	keys: make(map[string]*ecdsa.PublicKey),
}

const jwksCacheTTL = 5 * time.Minute

// debugLog prints debug messages if debug mode is enabled
func debugLog(config AuthConfig, format string, args ...interface{}) {
	if config.Debug {
		fmt.Printf("[AUTH DEBUG] "+format+"\n", args...)
	}
}

// JWKS represents the JSON Web Key Set response
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a single JSON Web Key
type JWK struct {
	Kty string `json:"kty"` // Key type (EC for ECDSA)
	Crv string `json:"crv"` // Curve (P-256 for ES256)
	X   string `json:"x"`   // X coordinate (base64url)
	Y   string `json:"y"`   // Y coordinate (base64url)
	Kid string `json:"kid"` // Key ID
	Alg string `json:"alg"` // Algorithm
	Use string `json:"use"` // Key use (sig for signature)
}

// fetchJWKS fetches the JWKS from the given URL
func fetchJWKS(url string, debug bool) (*JWKS, error) {
	if debug {
		fmt.Printf("[AUTH DEBUG] Fetching JWKS from: %s\n", url)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("JWKS fetch returned status %d: %s", resp.StatusCode, string(body))
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	if debug {
		fmt.Printf("[AUTH DEBUG] Fetched %d keys from JWKS\n", len(jwks.Keys))
	}

	return &jwks, nil
}

// getPublicKey retrieves the public key for the given kid from cache or fetches it
func getPublicKey(config AuthConfig, kid string) (*ecdsa.PublicKey, error) {
	// Check cache first
	globalJWKSCache.mu.RLock()
	if key, ok := globalJWKSCache.keys[kid]; ok && time.Since(globalJWKSCache.fetchedAt) < jwksCacheTTL {
		globalJWKSCache.mu.RUnlock()
		debugLog(config, "Using cached public key for kid: %s", kid)
		return key, nil
	}
	globalJWKSCache.mu.RUnlock()

	// Fetch fresh JWKS
	jwks, err := fetchJWKS(config.JWKSURL, config.Debug)
	if err != nil {
		return nil, err
	}

	// Update cache
	globalJWKSCache.mu.Lock()
	defer globalJWKSCache.mu.Unlock()

	globalJWKSCache.keys = make(map[string]*ecdsa.PublicKey)
	globalJWKSCache.fetchedAt = time.Now()

	for _, jwk := range jwks.Keys {
		if jwk.Kty != "EC" || jwk.Crv != "P-256" {
			debugLog(config, "Skipping non-EC P-256 key: kid=%s, kty=%s, crv=%s", jwk.Kid, jwk.Kty, jwk.Crv)
			continue
		}

		pubKey, err := parseECPublicKey(jwk)
		if err != nil {
			debugLog(config, "Failed to parse key kid=%s: %v", jwk.Kid, err)
			continue
		}

		globalJWKSCache.keys[jwk.Kid] = pubKey
		debugLog(config, "Cached public key for kid: %s", jwk.Kid)
	}

	if key, ok := globalJWKSCache.keys[kid]; ok {
		return key, nil
	}

	return nil, fmt.Errorf("key with kid %s not found in JWKS", kid)
}

// parseECPublicKey converts a JWK to an ecdsa.PublicKey
func parseECPublicKey(jwk JWK) (*ecdsa.PublicKey, error) {
	// Decode base64url-encoded X and Y coordinates
	xBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		return nil, fmt.Errorf("failed to decode X coordinate: %w", err)
	}

	yBytes, err := base64.RawURLEncoding.DecodeString(jwk.Y)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Y coordinate: %w", err)
	}

	// Create the public key with P-256 curve
	pubKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}

	return pubKey, nil
}

// NewAuthMiddleware creates a new JWT authentication middleware
func NewAuthMiddleware(config AuthConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			debugLog(config, "Missing authorization header")
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Error:   "unauthorized",
				Message: "Missing authorization header",
				Code:    401,
			})
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			debugLog(config, "Invalid authorization header format")
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid authorization header format",
				Code:    401,
			})
		}

		tokenString := parts[1]
		debugLog(config, "Token received (first 20 chars): %s...", tokenString[:min(20, len(tokenString))])

		// Build parser options for claim validation
		parserOpts := []jwt.ParserOption{}
		if config.Audience != "" {
			parserOpts = append(parserOpts, jwt.WithAudience(config.Audience))
			debugLog(config, "Validating audience claim: %s", config.Audience)
		}
		if config.Issuer != "" {
			parserOpts = append(parserOpts, jwt.WithIssuer(config.Issuer))
			debugLog(config, "Validating issuer claim: %s", config.Issuer)
		}

		// Create key function that supports both HMAC and ECDSA
		keyFunc := func(token *jwt.Token) (interface{}, error) {
			switch token.Method.(type) {
			case *jwt.SigningMethodECDSA:
				// ES256 - need public key from JWKS
				if config.JWKSURL == "" {
					debugLog(config, "ECDSA token but no JWKS URL configured")
					return nil, fmt.Errorf("ECDSA signing requires JWKS URL configuration")
				}

				kid, ok := token.Header["kid"].(string)
				if !ok {
					debugLog(config, "Token missing 'kid' header for ECDSA verification")
					return nil, fmt.Errorf("token missing kid header")
				}

				debugLog(config, "Looking up public key for kid: %s", kid)
				return getPublicKey(config, kid)

			case *jwt.SigningMethodHMAC:
				// HS256 - use shared secret
				if config.JWTSecret == "" {
					debugLog(config, "HMAC token but no JWT secret configured")
					return nil, fmt.Errorf("HMAC signing requires JWT secret configuration")
				}
				return []byte(config.JWTSecret), nil

			default:
				debugLog(config, "Unsupported signing method: %v", token.Method.Alg())
				return nil, jwt.ErrSignatureInvalid
			}
		}

		// Parse and validate the token
		token, err := jwt.Parse(tokenString, keyFunc, parserOpts...)

		if err != nil {
			debugLog(config, "Token validation failed: %v", err)
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid or expired token",
				Code:    401,
			})
		}

		if !token.Valid {
			debugLog(config, "Token marked as invalid")
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid or expired token",
				Code:    401,
			})
		}

		// Extract claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			debugLog(config, "Failed to extract claims from token")
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid token claims",
				Code:    401,
			})
		}

		// Get user ID from the "sub" claim (Supabase standard)
		userID, ok := claims["sub"].(string)
		if !ok || userID == "" {
			debugLog(config, "Missing or invalid 'sub' claim in token")
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Error:   "unauthorized",
				Message: "Missing user ID in token",
				Code:    401,
			})
		}

		debugLog(config, "Authentication successful for user: %s", userID)

		// Store user ID in context for handlers to use
		c.Locals(UserIDKey, userID)

		return c.Next()
	}
}

// OptionalAuthMiddleware creates middleware that extracts user ID if present but doesn't require auth
func OptionalAuthMiddleware(config AuthConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			debugLog(config, "[Optional] No authorization header present")
			return c.Next()
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			debugLog(config, "[Optional] Invalid authorization header format, continuing without auth")
			return c.Next()
		}

		tokenString := parts[1]
		debugLog(config, "[Optional] Token received (first 20 chars): %s...", tokenString[:min(20, len(tokenString))])

		// Build parser options for claim validation
		parserOpts := []jwt.ParserOption{}
		if config.Audience != "" {
			parserOpts = append(parserOpts, jwt.WithAudience(config.Audience))
			debugLog(config, "[Optional] Validating audience claim: %s", config.Audience)
		}
		if config.Issuer != "" {
			parserOpts = append(parserOpts, jwt.WithIssuer(config.Issuer))
			debugLog(config, "[Optional] Validating issuer claim: %s", config.Issuer)
		}

		// Create key function that supports both HMAC and ECDSA
		keyFunc := func(token *jwt.Token) (interface{}, error) {
			switch token.Method.(type) {
			case *jwt.SigningMethodECDSA:
				if config.JWKSURL == "" {
					debugLog(config, "[Optional] ECDSA token but no JWKS URL configured")
					return nil, fmt.Errorf("ECDSA signing requires JWKS URL configuration")
				}

				kid, ok := token.Header["kid"].(string)
				if !ok {
					debugLog(config, "[Optional] Token missing 'kid' header for ECDSA verification")
					return nil, fmt.Errorf("token missing kid header")
				}

				debugLog(config, "[Optional] Looking up public key for kid: %s", kid)
				return getPublicKey(config, kid)

			case *jwt.SigningMethodHMAC:
				if config.JWTSecret == "" {
					debugLog(config, "[Optional] HMAC token but no JWT secret configured")
					return nil, fmt.Errorf("HMAC signing requires JWT secret configuration")
				}
				return []byte(config.JWTSecret), nil

			default:
				debugLog(config, "[Optional] Unsupported signing method: %v", token.Method.Alg())
				return nil, jwt.ErrSignatureInvalid
			}
		}

		token, err := jwt.Parse(tokenString, keyFunc, parserOpts...)

		if err != nil {
			debugLog(config, "[Optional] Token validation failed: %v, continuing without auth", err)
			return c.Next()
		}

		if !token.Valid {
			debugLog(config, "[Optional] Token marked as invalid, continuing without auth")
			return c.Next()
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			debugLog(config, "[Optional] Failed to extract claims, continuing without auth")
			return c.Next()
		}

		userID, ok := claims["sub"].(string)
		if ok && userID != "" {
			debugLog(config, "[Optional] Authentication successful for user: %s", userID)
			c.Locals(UserIDKey, userID)
		} else {
			debugLog(config, "[Optional] Missing or invalid 'sub' claim, continuing without auth")
		}

		return c.Next()
	}
}

// GetUserID extracts the user ID from the fiber context
func GetUserID(c *fiber.Ctx) string {
	userID, ok := c.Locals(UserIDKey).(string)
	if !ok {
		return ""
	}
	return userID
}

// RequireUserID is a helper that returns an error response if no user ID is present
func RequireUserID(c *fiber.Ctx) (string, error) {
	userID := GetUserID(c)
	if userID == "" {
		return "", c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
			Code:    401,
		})
	}
	return userID, nil
}
