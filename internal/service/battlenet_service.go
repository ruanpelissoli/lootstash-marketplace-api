package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/cache"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
)

const (
	battleNetStateTTL    = 10 * time.Minute
	battleNetStatePrefix = "battlenet:state:"
)

// BattleNetConfig holds Battle.net OAuth configuration
type BattleNetConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// BattleNetService handles Battle.net OAuth operations
type BattleNetService struct {
	config      BattleNetConfig
	redis       *cache.RedisClient
	profileRepo repository.ProfileRepository
	invalidator *cache.Invalidator
}

// NewBattleNetService creates a new Battle.net service
func NewBattleNetService(config BattleNetConfig, redis *cache.RedisClient, profileRepo repository.ProfileRepository) *BattleNetService {
	return &BattleNetService{
		config:      config,
		redis:       redis,
		profileRepo: profileRepo,
		invalidator: cache.NewInvalidator(redis),
	}
}

// battleNetTokenResponse represents the OAuth token response
type battleNetTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// battleNetUserInfo represents the userinfo response
type battleNetUserInfo struct {
	ID        int64  `json:"id"`
	BattleTag string `json:"battletag"`
}

// GetAuthorizationURL generates an OAuth authorization URL and stores state in Redis
func (s *BattleNetService) GetAuthorizationURL(ctx context.Context, userID string, region string) (string, error) {
	// Redis is required for OAuth state management
	if s.redis == nil || !s.redis.IsAvailable() {
		return "", fmt.Errorf("Battle.net linking is temporarily unavailable")
	}

	// Default to US region if not specified
	if region == "" {
		region = "us"
	}

	// Generate secure random state
	state, err := generateState()
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	// Store state with userID in Redis
	stateKey := battleNetStatePrefix + state
	stateData := fmt.Sprintf("%s:%s", userID, region)
	if err := s.redis.Set(ctx, stateKey, stateData, battleNetStateTTL); err != nil {
		return "", fmt.Errorf("failed to store state: %w", err)
	}

	// Build authorization URL
	authURL := fmt.Sprintf("https://%s.battle.net/oauth/authorize", region)
	params := url.Values{
		"client_id":     {s.config.ClientID},
		"redirect_uri":  {s.config.RedirectURI},
		"response_type": {"code"},
		"scope":         {"openid"},
		"state":         {state},
	}

	return authURL + "?" + params.Encode(), nil
}

// HandleCallback processes the OAuth callback and links the Battle.net account
func (s *BattleNetService) HandleCallback(ctx context.Context, userID string, code string, state string) (*models.Profile, error) {
	// Redis is required for OAuth state validation
	if s.redis == nil || !s.redis.IsAvailable() {
		return nil, fmt.Errorf("Battle.net linking is temporarily unavailable")
	}

	// Validate and consume state from Redis (one-time use)
	stateKey := battleNetStatePrefix + state
	stateData, err := s.redis.Get(ctx, stateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired state")
	}

	// Delete state immediately (one-time use)
	_ = s.redis.Del(ctx, stateKey)

	// Parse state data (userID:region)
	parts := strings.SplitN(stateData, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid state data")
	}
	storedUserID, region := parts[0], parts[1]

	// Verify userID matches
	if storedUserID != userID {
		return nil, fmt.Errorf("state mismatch")
	}

	// Exchange code for access token
	token, err := s.exchangeCodeForToken(ctx, code, region)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Fetch user info from Battle.net
	userInfo, err := s.fetchUserInfo(ctx, token, region)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}

	// Get current profile
	profile, err := s.profileRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	// Update profile with Battle.net info
	now := time.Now()
	profile.BattleNetID = &userInfo.ID
	profile.BattleTag = &userInfo.BattleTag
	profile.BattleNetLinkedAt = &now

	if err := s.profileRepo.Update(ctx, profile); err != nil {
		// Check for unique constraint violation (duplicate Battle.net ID)
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return nil, fmt.Errorf("this Battle.net account is already linked to another user")
		}
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	// Invalidate profile cache
	_ = s.invalidator.InvalidateProfile(ctx, userID)
	_ = s.invalidator.InvalidateProfileDTO(ctx, userID)

	return profile, nil
}

// Unlink removes Battle.net account link from user profile
func (s *BattleNetService) Unlink(ctx context.Context, userID string) error {
	// Get current profile
	profile, err := s.profileRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get profile: %w", err)
	}

	// Check if Battle.net is linked
	if !profile.IsBattleNetLinked() {
		return fmt.Errorf("no Battle.net account linked")
	}

	// Clear Battle.net fields
	profile.BattleNetID = nil
	profile.BattleTag = nil
	profile.BattleNetLinkedAt = nil

	if err := s.profileRepo.Update(ctx, profile); err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	// Invalidate profile cache
	_ = s.invalidator.InvalidateProfile(ctx, userID)
	_ = s.invalidator.InvalidateProfileDTO(ctx, userID)

	return nil
}

// exchangeCodeForToken exchanges the authorization code for an access token
func (s *BattleNetService) exchangeCodeForToken(ctx context.Context, code string, region string) (string, error) {
	tokenURL := fmt.Sprintf("https://%s.battle.net/oauth/token", region)

	data := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {s.config.RedirectURI},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	req.SetBasicAuth(s.config.ClientID, s.config.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token exchange failed: %s", string(body))
	}

	var tokenResp battleNetTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	return tokenResp.AccessToken, nil
}

// fetchUserInfo fetches the user's Battle.net info using the access token
func (s *BattleNetService) fetchUserInfo(ctx context.Context, accessToken string, region string) (*battleNetUserInfo, error) {
	userInfoURL := fmt.Sprintf("https://%s.battle.net/oauth/userinfo", region)

	req, err := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("userinfo request failed: %s", string(body))
	}

	var userInfo battleNetUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// generateState generates a cryptographically secure random state string
func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
