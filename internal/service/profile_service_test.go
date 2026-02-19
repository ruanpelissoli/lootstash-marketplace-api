package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/cache"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository/mocks"
	storageMocks "github.com/ruanpelissoli/lootstash-marketplace-api/internal/storage/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestProfileGetByID_Success(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	svc := NewProfileService(profileRepo, newTestRedis(), nil)

	ctx := context.Background()
	profile := testProfile(testUserID)

	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil)

	result, err := svc.GetByID(ctx, testUserID)
	assert.NoError(t, err)
	assert.Equal(t, testUserID, result.ID)
	assert.Equal(t, profile.Username, result.Username)

	profileRepo.AssertExpectations(t)
}

func TestProfileGetByID_CachesResult(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	redisClient, mr := newTestRedisReal(t)
	svc := NewProfileService(profileRepo, redisClient, nil)

	ctx := context.Background()
	profile := testProfile(testUserID)

	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil)

	_, err := svc.GetByID(ctx, testUserID)
	assert.NoError(t, err)

	// Verify profile:{id} key was written to Redis
	cacheKey := cache.ProfileKey(testUserID)
	assert.True(t, mr.Exists(cacheKey))

	val, err := mr.Get(cacheKey)
	assert.NoError(t, err)

	var cachedProfile models.Profile
	err = json.Unmarshal([]byte(val), &cachedProfile)
	assert.NoError(t, err)
	assert.Equal(t, testUserID, cachedProfile.ID)

	profileRepo.AssertExpectations(t)
}

func TestProfileGetByID_CachesDTO(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	redisClient, mr := newTestRedisReal(t)
	svc := NewProfileService(profileRepo, redisClient, nil)

	ctx := context.Background()
	profile := testProfile(testUserID)

	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil)

	_, err := svc.GetByID(ctx, testUserID)
	assert.NoError(t, err)

	// Verify profile:dto:{id} key was written to Redis
	dtoCacheKey := cache.ProfileDTOKey(testUserID)
	assert.True(t, mr.Exists(dtoCacheKey))

	val, err := mr.Get(dtoCacheKey)
	assert.NoError(t, err)

	var cachedDTO dto.ProfileResponse
	err = json.Unmarshal([]byte(val), &cachedDTO)
	assert.NoError(t, err)
	assert.Equal(t, testUserID, cachedDTO.ID)

	profileRepo.AssertExpectations(t)
}

func TestProfileGetByID_CacheHit(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	redisClient, mr := newTestRedisReal(t)
	svc := NewProfileService(profileRepo, redisClient, nil)

	ctx := context.Background()
	profile := testProfile(testUserID)

	// Pre-populate miniredis with cached profile JSON
	data, err := json.Marshal(profile)
	assert.NoError(t, err)
	mr.Set(cache.ProfileKey(testUserID), string(data))

	result, err := svc.GetByID(ctx, testUserID)
	assert.NoError(t, err)
	assert.Equal(t, testUserID, result.ID)
	assert.Equal(t, profile.Username, result.Username)

	// Repo should NOT have been called
	profileRepo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
}

// ---------------------------------------------------------------------------
// GetByUsername
// ---------------------------------------------------------------------------

func TestProfileGetByUsername_Success(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	svc := NewProfileService(profileRepo, newTestRedis(), nil)

	ctx := context.Background()
	profile := testProfile(testUserID)

	profileRepo.On("GetByUsername", ctx, profile.Username).Return(profile, nil)

	result, err := svc.GetByUsername(ctx, profile.Username)
	assert.NoError(t, err)
	assert.Equal(t, testUserID, result.ID)
	assert.Equal(t, profile.Username, result.Username)

	profileRepo.AssertExpectations(t)
}

func TestProfileGetByUsername_CachesResult(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	redisClient, mr := newTestRedisReal(t)
	svc := NewProfileService(profileRepo, redisClient, nil)

	ctx := context.Background()
	profile := testProfile(testUserID)
	username := profile.Username

	profileRepo.On("GetByUsername", ctx, username).Return(profile, nil)

	_, err := svc.GetByUsername(ctx, username)
	assert.NoError(t, err)

	// Verify profile:username:{name} key was written (lowercase)
	cacheKey := cache.ProfileUsernameKey(username)
	assert.True(t, mr.Exists(cacheKey))

	val, err := mr.Get(cacheKey)
	assert.NoError(t, err)

	var cachedProfile models.Profile
	err = json.Unmarshal([]byte(val), &cachedProfile)
	assert.NoError(t, err)
	assert.Equal(t, testUserID, cachedProfile.ID)

	profileRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// GetByIdentifier
// ---------------------------------------------------------------------------

func TestProfileGetByIdentifier_UUID(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	svc := NewProfileService(profileRepo, newTestRedis(), nil)

	ctx := context.Background()
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	profile := testProfile(validUUID)

	profileRepo.On("GetByID", ctx, validUUID).Return(profile, nil)

	result, err := svc.GetByIdentifier(ctx, validUUID)
	assert.NoError(t, err)
	assert.Equal(t, validUUID, result.ID)

	// GetByID should have been called, not GetByUsername
	profileRepo.AssertCalled(t, "GetByID", ctx, validUUID)
	profileRepo.AssertNotCalled(t, "GetByUsername", mock.Anything, mock.Anything)
}

func TestProfileGetByIdentifier_Username(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	svc := NewProfileService(profileRepo, newTestRedis(), nil)

	ctx := context.Background()
	username := "cooltrader42"
	profile := testProfile(testUserID)
	profile.Username = username

	profileRepo.On("GetByUsername", ctx, username).Return(profile, nil)

	result, err := svc.GetByIdentifier(ctx, username)
	assert.NoError(t, err)
	assert.Equal(t, username, result.Username)

	// GetByUsername should have been called, not GetByID
	profileRepo.AssertCalled(t, "GetByUsername", ctx, username)
	profileRepo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestProfileUpdate_Success(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	svc := NewProfileService(profileRepo, newTestRedis(), nil)

	ctx := context.Background()
	profile := testProfile(testUserID)

	newDisplayName := "New Display Name"
	newTimezone := "America/New_York"
	req := &dto.UpdateProfileRequest{
		DisplayName: &newDisplayName,
		Timezone:    &newTimezone,
	}

	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil)
	profileRepo.On("Update", ctx, profile).Return(nil)

	result, err := svc.Update(ctx, testUserID, req)
	assert.NoError(t, err)
	assert.Equal(t, &newDisplayName, result.DisplayName)
	assert.Equal(t, &newTimezone, result.Timezone)

	profileRepo.AssertExpectations(t)
}

func TestProfileUpdate_InvalidatesAllCaches(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	redisClient, mr := newTestRedisReal(t)
	svc := NewProfileService(profileRepo, redisClient, nil)

	ctx := context.Background()
	profile := testProfile(testUserID)

	// Pre-set all cache keys
	mr.Set(cache.ProfileKey(testUserID), "cached")
	mr.Set(cache.ProfileDTOKey(testUserID), "cached")
	mr.Set(cache.ProfileUsernameKey(profile.Username), "cached")

	// Verify they all exist
	assert.True(t, mr.Exists(cache.ProfileKey(testUserID)))
	assert.True(t, mr.Exists(cache.ProfileDTOKey(testUserID)))
	assert.True(t, mr.Exists(cache.ProfileUsernameKey(profile.Username)))

	newDisplayName := "Updated Name"
	req := &dto.UpdateProfileRequest{
		DisplayName: &newDisplayName,
	}

	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil)
	profileRepo.On("Update", ctx, profile).Return(nil)

	_, err := svc.Update(ctx, testUserID, req)
	assert.NoError(t, err)

	// Verify all cache keys were invalidated
	assert.False(t, mr.Exists(cache.ProfileKey(testUserID)))
	assert.False(t, mr.Exists(cache.ProfileDTOKey(testUserID)))
	assert.False(t, mr.Exists(cache.ProfileUsernameKey(profile.Username)))

	profileRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// IsAdmin
// ---------------------------------------------------------------------------

func TestProfileIsAdmin_True(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	svc := NewProfileService(profileRepo, newTestRedis(), nil)

	ctx := context.Background()
	profile := testProfile(testUserID, withAdmin)

	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil)

	isAdmin, err := svc.IsAdmin(ctx, testUserID)
	assert.NoError(t, err)
	assert.True(t, isAdmin)

	profileRepo.AssertExpectations(t)
}

func TestProfileIsAdmin_False(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	svc := NewProfileService(profileRepo, newTestRedis(), nil)

	ctx := context.Background()
	profile := testProfile(testUserID) // not admin by default

	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil)

	isAdmin, err := svc.IsAdmin(ctx, testUserID)
	assert.NoError(t, err)
	assert.False(t, isAdmin)

	profileRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// UploadProfilePicture
// ---------------------------------------------------------------------------

func TestUploadProfilePicture_Success(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	stor := new(storageMocks.MockStorage)
	svc := NewProfileService(profileRepo, newTestRedis(), stor)

	ctx := context.Background()
	profile := testProfile(testUserID)
	imageData := []byte("fake-image-data")
	contentType := "image/png"
	expectedURL := "https://storage.example.com/user-111.png"

	stor.On("UploadImage", ctx, testUserID+".png", imageData, contentType).Return(expectedURL, nil)
	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil)
	profileRepo.On("Update", ctx, profile).Return(nil)

	url, err := svc.UploadProfilePicture(ctx, testUserID, imageData, contentType)
	assert.NoError(t, err)
	assert.Equal(t, expectedURL, url)
	assert.Equal(t, &expectedURL, profile.AvatarURL)

	stor.AssertExpectations(t)
	profileRepo.AssertExpectations(t)
}

func TestUploadProfilePicture_UnsupportedContentType(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	stor := new(storageMocks.MockStorage)
	svc := NewProfileService(profileRepo, newTestRedis(), stor)

	ctx := context.Background()
	imageData := []byte("fake-gif-data")

	_, err := svc.UploadProfilePicture(ctx, testUserID, imageData, "image/gif")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported content type")

	// Storage and repo should not have been called
	stor.AssertNotCalled(t, "UploadImage", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	profileRepo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
}

func TestUploadProfilePicture_NoStorage(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	svc := NewProfileService(profileRepo, newTestRedis(), nil) // nil storage

	ctx := context.Background()
	imageData := []byte("fake-image-data")

	_, err := svc.UploadProfilePicture(ctx, testUserID, imageData, "image/png")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage not configured")
}

// ---------------------------------------------------------------------------
// GetSales
// ---------------------------------------------------------------------------

func TestGetSales_Success(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	transactionRepo := new(mocks.MockTransactionRepository)
	svc := NewProfileService(profileRepo, newTestRedis(), nil)
	svc.SetTransactionRepository(transactionRepo)

	ctx := context.Background()
	completedAt := time.Now().Add(-1 * time.Hour)

	records := []repository.SaleRecord{
		{
			TransactionID: testTransactionID,
			CompletedAt:   completedAt,
			ItemName:      "Shako",
			ItemType:      "unique",
			Rarity:        "unique",
			ImageURL:      strPtr("https://example.com/shako.png"),
			BaseName:      strPtr("Harlequin Crest"),
			Stats:         json.RawMessage(`[{"code":"ac%","value":163,"displayText":"+163% Enhanced Defense","isVariable":true}]`),
			OfferedItems:  json.RawMessage(`[{"type":"rune","name":"Ist Rune","quantity":2}]`),
			BuyerID:       testBuyerID,
			BuyerName:     "BuyerUser",
			BuyerAvatar:   strPtr("https://example.com/buyer.png"),
			ReviewRating:  intPtr(5),
			ReviewComment: strPtr("Great trade!"),
			ReviewedAt:    completedAt.Add(10 * time.Minute),
		},
	}

	transactionRepo.On("GetSalesBySeller", ctx, testSellerID, 0, 10).Return(records, 1, nil)

	result, err := svc.GetSales(ctx, testSellerID, 0, 10)
	assert.NoError(t, err)
	assert.Len(t, result.Sales, 1)
	assert.Equal(t, 1, result.Total)
	assert.False(t, result.HasMore)

	sale := result.Sales[0]
	assert.Equal(t, testTransactionID, sale.ID)
	assert.Equal(t, "Shako", sale.Item.Name)
	assert.Equal(t, "unique", sale.Item.ItemType)
	assert.Equal(t, "Harlequin Crest", sale.Item.BaseName)
	assert.Len(t, sale.Item.Stats, 1)
	assert.Equal(t, "ac%", sale.Item.Stats[0].Code)
	assert.Equal(t, "+163% Enhanced Defense", sale.Item.Stats[0].DisplayText)
	assert.Len(t, sale.SoldFor, 1)
	assert.Equal(t, "Ist Rune", sale.SoldFor[0].Name)
	assert.Equal(t, 2, sale.SoldFor[0].Quantity)
	assert.Equal(t, testBuyerID, sale.Buyer.ID)
	assert.NotNil(t, sale.Review)
	assert.Equal(t, 5, sale.Review.Rating)
	assert.Equal(t, "Great trade!", sale.Review.Comment)

	transactionRepo.AssertExpectations(t)
}

func TestGetSales_NoTransactionRepo(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	svc := NewProfileService(profileRepo, newTestRedis(), nil)
	// Do NOT call SetTransactionRepository

	ctx := context.Background()

	result, err := svc.GetSales(ctx, testSellerID, 0, 10)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "transaction repository not configured")
}

// ---------------------------------------------------------------------------
// DTO transformations
// ---------------------------------------------------------------------------

func TestProfileToResponse_MapsAllFields(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	svc := NewProfileService(profileRepo, newTestRedis(), nil)

	battleTag := "Player#1234"
	flair := "gold"
	usernameColor := "#FFD700"
	timezone := "Europe/London"
	now := time.Now()

	profile := &models.Profile{
		ID:            testUserID,
		Username:      "trademaster",
		DisplayName:   strPtr("Trade Master"),
		AvatarURL:     strPtr("https://example.com/avatar.png"),
		BattleTag:     &battleTag,
		TotalTrades:   42,
		AverageRating: 4.8,
		RatingCount:   15,
		IsPremium:     true,
		IsAdmin:       false,
		ProfileFlair:  &flair,
		UsernameColor: &usernameColor,
		Timezone:      &timezone,
		CreatedAt:     now,
	}

	resp := svc.ToResponse(profile)

	assert.Equal(t, testUserID, resp.ID)
	assert.Equal(t, "trademaster", resp.Username)
	assert.Equal(t, "Trade Master", resp.DisplayName)
	assert.Equal(t, "https://example.com/avatar.png", resp.AvatarURL)
	assert.Equal(t, "Player#1234", resp.BattleTag)
	assert.Equal(t, 42, resp.TotalTrades)
	assert.Equal(t, 4.8, resp.AverageRating)
	assert.Equal(t, 15, resp.RatingCount)
	assert.True(t, resp.IsPremium)
	assert.False(t, resp.IsAdmin)
	assert.Equal(t, "gold", resp.ProfileFlair)
	assert.Equal(t, "#FFD700", resp.UsernameColor)
	assert.Equal(t, "Europe/London", resp.Timezone)
	assert.Equal(t, now, resp.CreatedAt)
}

func TestProfileToMyProfileResponse_IncludesBattleNetFields(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	svc := NewProfileService(profileRepo, newTestRedis(), nil)

	battleNetID := int64(12345)
	linkedAt := time.Now().Add(-24 * time.Hour)
	updatedAt := time.Now()

	profile := testProfile(testUserID)
	profile.BattleNetID = &battleNetID
	profile.BattleNetLinkedAt = &linkedAt
	profile.UpdatedAt = updatedAt

	resp := svc.ToMyProfileResponse(profile)

	assert.Equal(t, testUserID, resp.ID)
	assert.True(t, resp.BattleNetLinked)
	assert.NotNil(t, resp.BattleNetLinkedAt)
	assert.Equal(t, linkedAt, *resp.BattleNetLinkedAt)
	assert.Equal(t, updatedAt, resp.UpdatedAt)
}

func TestTransformSaleStats_WithDisplayText(t *testing.T) {
	svc := NewProfileService(nil, newTestRedis(), nil)

	rawStats := json.RawMessage(`[
		{"code":"ac%","value":163,"displayText":"+163% Enhanced Defense","isVariable":true},
		{"code":"str","value":30,"displayText":"+30 to Strength","isVariable":false}
	]`)

	result := svc.transformSaleStats(rawStats)

	assert.Len(t, result, 2)

	assert.Equal(t, "ac%", result[0].Code)
	assert.NotNil(t, result[0].Value)
	assert.Equal(t, 163, *result[0].Value)
	assert.Equal(t, "+163% Enhanced Defense", result[0].DisplayText)
	assert.True(t, result[0].IsVariable)

	assert.Equal(t, "str", result[1].Code)
	assert.NotNil(t, result[1].Value)
	assert.Equal(t, 30, *result[1].Value)
	assert.Equal(t, "+30 to Strength", result[1].DisplayText)
	assert.False(t, result[1].IsVariable)
}

func TestTransformSaleStats_FallbackDisplayText(t *testing.T) {
	svc := NewProfileService(nil, newTestRedis(), nil)

	// Empty displayText should produce "code: value" fallback
	rawStats := json.RawMessage(`[
		{"code":"ac%","value":100,"displayText":"","isVariable":true},
		{"code":"indestructible","displayText":"","isVariable":false}
	]`)

	result := svc.transformSaleStats(rawStats)

	assert.Len(t, result, 2)

	// With numeric value: "code: value"
	assert.Equal(t, "ac%: 100", result[0].DisplayText)

	// Without value: just code
	assert.Equal(t, "indestructible", result[1].DisplayText)
}

func TestTransformOfferedItems_ZeroQuantity_DefaultsToOne(t *testing.T) {
	svc := NewProfileService(nil, newTestRedis(), nil)

	rawItems := json.RawMessage(`[
		{"type":"rune","name":"Ist Rune","quantity":0,"imageUrl":"https://example.com/ist.png"},
		{"type":"rune","name":"Ber Rune","quantity":3}
	]`)

	result := svc.transformOfferedItems(rawItems)

	assert.Len(t, result, 2)

	// Zero quantity defaults to 1
	assert.Equal(t, "Ist Rune", result[0].Name)
	assert.Equal(t, 1, result[0].Quantity)
	assert.Equal(t, "https://example.com/ist.png", result[0].ImageURL)

	// Non-zero quantity preserved
	assert.Equal(t, "Ber Rune", result[1].Name)
	assert.Equal(t, 3, result[1].Quantity)
}

func TestExtractSaleNumericValue(t *testing.T) {
	// float64 input (common from JSON unmarshal)
	result := extractSaleNumericValue(float64(42))
	assert.NotNil(t, result)
	assert.Equal(t, 42, *result)

	// int input
	result = extractSaleNumericValue(int(99))
	assert.NotNil(t, result)
	assert.Equal(t, 99, *result)

	// string input with number
	result = extractSaleNumericValue("+163")
	assert.NotNil(t, result)
	assert.Equal(t, 163, *result)

	// string input with negative number
	result = extractSaleNumericValue("-25")
	assert.NotNil(t, result)
	assert.Equal(t, -25, *result)

	// nil input
	result = extractSaleNumericValue(nil)
	assert.Nil(t, result)

	// string with no digits
	result = extractSaleNumericValue("no-digits")
	assert.Nil(t, result)
}
