package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/cache"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository/mocks"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func setupListingService(profileRepo *mocks.MockProfileRepository, listingRepo *mocks.MockListingRepository, redis *cache.RedisClient) (*ListingService, *ProfileService) {
	profileService := NewProfileService(profileRepo, redis, nil)
	svc := NewListingService(listingRepo, profileService, redis)
	return svc, profileService
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestListingCreate_PremiumUser_Success(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	profile := testProfile(testSellerID, withPremium)
	profileRepo.On("GetByID", mock.Anything, testSellerID).Return(profile, nil)
	listingRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Listing")).Return(nil)

	req := &dto.CreateListingRequest{
		Name:      "Shako",
		ItemType:  "unique",
		Rarity:    "unique",
		Category:  "helm",
		Game:      "diablo2",
		Platforms: []string{"pc"},
		Region:    "americas",
	}

	listing, err := svc.Create(context.Background(), testSellerID, req)

	assert.NoError(t, err)
	assert.NotNil(t, listing)
	assert.Equal(t, testSellerID, listing.SellerID)
	assert.Equal(t, "Shako", listing.Name)
	assert.Equal(t, "active", listing.Status)
	// Premium user: CountActiveBySellerID should NOT have been called
	listingRepo.AssertNotCalled(t, "CountActiveBySellerID", mock.Anything, mock.Anything)
	profileRepo.AssertExpectations(t)
	listingRepo.AssertExpectations(t)
}

func TestListingCreate_FreeUser_UnderLimit(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	profile := testProfile(testSellerID) // free user
	profileRepo.On("GetByID", mock.Anything, testSellerID).Return(profile, nil)
	listingRepo.On("CountActiveBySellerID", mock.Anything, testSellerID).Return(5, nil)
	listingRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Listing")).Return(nil)

	req := &dto.CreateListingRequest{
		Name:      "War Traveler",
		ItemType:  "unique",
		Rarity:    "unique",
		Category:  "boots",
		Game:      "diablo2",
		Platforms: []string{"pc"},
		Region:    "americas",
	}

	listing, err := svc.Create(context.Background(), testSellerID, req)

	assert.NoError(t, err)
	assert.NotNil(t, listing)
	assert.Equal(t, "active", listing.Status)
	profileRepo.AssertExpectations(t)
	listingRepo.AssertExpectations(t)
}

func TestListingCreate_FreeUser_AtLimit(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	profile := testProfile(testSellerID) // free user
	profileRepo.On("GetByID", mock.Anything, testSellerID).Return(profile, nil)
	listingRepo.On("CountActiveBySellerID", mock.Anything, testSellerID).Return(FreeListingLimit, nil)

	req := &dto.CreateListingRequest{
		Name:      "Shako",
		ItemType:  "unique",
		Rarity:    "unique",
		Category:  "helm",
		Game:      "diablo2",
		Platforms: []string{"pc"},
		Region:    "americas",
	}

	listing, err := svc.Create(context.Background(), testSellerID, req)

	assert.ErrorIs(t, err, ErrListingLimitReached)
	assert.Nil(t, listing)
	listingRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	profileRepo.AssertExpectations(t)
	listingRepo.AssertExpectations(t)
}

func TestListingCreate_DeduplicatesPlatforms(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	profile := testProfile(testSellerID, withPremium)
	profileRepo.On("GetByID", mock.Anything, testSellerID).Return(profile, nil)

	var capturedListing *models.Listing
	listingRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Listing")).
		Run(func(args mock.Arguments) {
			capturedListing = args.Get(1).(*models.Listing)
		}).Return(nil)

	req := &dto.CreateListingRequest{
		Name:      "Enigma",
		ItemType:  "runeword",
		Rarity:    "runeword",
		Category:  "body armor",
		Game:      "diablo2",
		Platforms: []string{"pc", "pc", "xbox"},
		Region:    "americas",
	}

	listing, err := svc.Create(context.Background(), testSellerID, req)

	assert.NoError(t, err)
	assert.NotNil(t, listing)
	assert.Equal(t, []string{"pc", "xbox"}, capturedListing.Platforms)
	profileRepo.AssertExpectations(t)
	listingRepo.AssertExpectations(t)
}

func TestListingCreate_SetsSellerTimezone(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	tz := "America/New_York"
	profile := testProfile(testSellerID, withPremium, withTimezone(tz))
	profileRepo.On("GetByID", mock.Anything, testSellerID).Return(profile, nil)

	var capturedListing *models.Listing
	listingRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Listing")).
		Run(func(args mock.Arguments) {
			capturedListing = args.Get(1).(*models.Listing)
		}).Return(nil)

	req := &dto.CreateListingRequest{
		Name:      "Arachnid Mesh",
		ItemType:  "unique",
		Rarity:    "unique",
		Category:  "belts",
		Game:      "diablo2",
		Platforms: []string{"pc"},
		Region:    "americas",
	}

	listing, err := svc.Create(context.Background(), testSellerID, req)

	assert.NoError(t, err)
	assert.NotNil(t, listing)
	assert.NotNil(t, capturedListing.SellerTimezone)
	assert.Equal(t, tz, *capturedListing.SellerTimezone)
	profileRepo.AssertExpectations(t)
	listingRepo.AssertExpectations(t)
}

func TestListingCreate_OptionalFieldsEmpty(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	profile := testProfile(testSellerID, withPremium)
	profileRepo.On("GetByID", mock.Anything, testSellerID).Return(profile, nil)

	var capturedListing *models.Listing
	listingRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Listing")).
		Run(func(args mock.Arguments) {
			capturedListing = args.Get(1).(*models.Listing)
		}).Return(nil)

	req := &dto.CreateListingRequest{
		Name:      "Tal Rasha Lid",
		ItemType:  "set",
		Rarity:    "set",
		Category:  "helm",
		Game:      "diablo2",
		Platforms: []string{"pc"},
		Region:    "americas",
		// ImageURL, Notes, AskingPrice all empty
	}

	_, err := svc.Create(context.Background(), testSellerID, req)

	assert.NoError(t, err)
	assert.Nil(t, capturedListing.ImageURL)
	assert.Nil(t, capturedListing.Notes)
	assert.Nil(t, capturedListing.AskingPrice)
	profileRepo.AssertExpectations(t)
	listingRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestListingGetByID_Success(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	expected := testListing(testListingID, testSellerID)
	listingRepo.On("GetByIDWithSeller", mock.Anything, testListingID).Return(expected, nil)

	listing, err := svc.GetByID(context.Background(), testListingID)

	assert.NoError(t, err)
	assert.Equal(t, testListingID, listing.ID)
	assert.Equal(t, testSellerID, listing.SellerID)
	listingRepo.AssertExpectations(t)
}

func TestListingGetByID_WritesToCache(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	redisClient, mr := newTestRedisReal(t)
	svc, _ := setupListingService(profileRepo, listingRepo, redisClient)

	expected := testListing(testListingID, testSellerID)
	listingRepo.On("GetByIDWithSeller", mock.Anything, testListingID).Return(expected, nil)

	_, err := svc.GetByID(context.Background(), testListingID)
	assert.NoError(t, err)

	// Verify listing:{id} key is set in Redis
	cacheKey := cache.ListingKey(testListingID)
	assert.True(t, mr.Exists(cacheKey), "listing cache key should exist after GetByID")

	listingRepo.AssertExpectations(t)
}

func TestListingGetByID_CachesDTO(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	redisClient, mr := newTestRedisReal(t)
	svc, _ := setupListingService(profileRepo, listingRepo, redisClient)

	expected := testListing(testListingID, testSellerID)
	listingRepo.On("GetByIDWithSeller", mock.Anything, testListingID).Return(expected, nil)

	_, err := svc.GetByID(context.Background(), testListingID)
	assert.NoError(t, err)

	// Verify listing:dto:{id} key is also set
	dtoCacheKey := cache.ListingDTOKey(testListingID)
	assert.True(t, mr.Exists(dtoCacheKey), "listing DTO cache key should exist after GetByID")

	listingRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestListingUpdate_Success(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	existing := testListing(testListingID, testSellerID)
	listingRepo.On("GetByID", mock.Anything, testListingID).Return(existing, nil)
	listingRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Listing")).Return(nil)

	newNotes := "Updated notes"
	req := &dto.UpdateListingRequest{
		Notes: &newNotes,
	}

	result, err := svc.Update(context.Background(), testListingID, testSellerID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, &newNotes, result.Notes)
	listingRepo.AssertExpectations(t)
}

func TestListingUpdate_NotOwner(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	existing := testListing(testListingID, testSellerID)
	listingRepo.On("GetByID", mock.Anything, testListingID).Return(existing, nil)

	req := &dto.UpdateListingRequest{}

	result, err := svc.Update(context.Background(), testListingID, "other-user-id", req)

	assert.ErrorIs(t, err, ErrForbidden)
	assert.Nil(t, result)
	listingRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	listingRepo.AssertExpectations(t)
}

func TestListingUpdate_InvalidatesCache(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	redisClient, mr := newTestRedisReal(t)
	svc, _ := setupListingService(profileRepo, listingRepo, redisClient)

	existing := testListing(testListingID, testSellerID)
	listingRepo.On("GetByID", mock.Anything, testListingID).Return(existing, nil)
	listingRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Listing")).Return(nil)

	// Pre-set cache keys
	listingKey := cache.ListingKey(testListingID)
	dtoKey := cache.ListingDTOKey(testListingID)
	mr.Set(listingKey, `{"id":"test"}`)
	mr.Set(dtoKey, `{"id":"test"}`)

	newNotes := "Updated notes"
	req := &dto.UpdateListingRequest{
		Notes: &newNotes,
	}

	_, err := svc.Update(context.Background(), testListingID, testSellerID, req)
	assert.NoError(t, err)

	// Verify cache keys are deleted
	assert.False(t, mr.Exists(listingKey), "listing cache key should be invalidated after update")
	assert.False(t, mr.Exists(dtoKey), "listing DTO cache key should be invalidated after update")

	listingRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestListingDelete_Success(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	existing := testListing(testListingID, testSellerID)
	listingRepo.On("GetByID", mock.Anything, testListingID).Return(existing, nil)
	listingRepo.On("Update", mock.Anything, mock.MatchedBy(func(l *models.Listing) bool {
		return l.Status == "cancelled"
	})).Return(nil)

	err := svc.Delete(context.Background(), testListingID, testSellerID)

	assert.NoError(t, err)
	listingRepo.AssertExpectations(t)
}

func TestListingDelete_NotOwner(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	existing := testListing(testListingID, testSellerID)
	listingRepo.On("GetByID", mock.Anything, testListingID).Return(existing, nil)

	err := svc.Delete(context.Background(), testListingID, "other-user-id")

	assert.ErrorIs(t, err, ErrForbidden)
	listingRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	listingRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Stats Transformation
// ---------------------------------------------------------------------------

func TestTransformCardStats_OnlyVariable(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	rawStats := json.RawMessage(`[
		{"code":"ac%","value":163,"displayText":"+163% Enhanced Defense","isVariable":true},
		{"code":"str","value":2,"displayText":"+2 to Strength","isVariable":false},
		{"code":"dr","value":10,"displayText":"Damage Reduced by 10%","isVariable":true}
	]`)

	result := svc.transformCardStats(rawStats)

	assert.Len(t, result, 2)
	assert.Equal(t, "ac%", result[0].Code)
	assert.True(t, result[0].IsVariable)
	assert.Equal(t, "dr", result[1].Code)
	assert.True(t, result[1].IsVariable)
}

func TestTransformCardStats_EmptyStats(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	// nil raw message
	result := svc.transformCardStats(nil)
	assert.Nil(t, result)

	// empty raw message
	result2 := svc.transformCardStats(json.RawMessage{})
	assert.Nil(t, result2)
}

func TestTransformCardStats_NoVariableStats(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	rawStats := json.RawMessage(`[
		{"code":"str","value":2,"displayText":"+2 to Strength","isVariable":false},
		{"code":"dex","value":5,"displayText":"+5 to Dexterity","isVariable":false}
	]`)

	result := svc.transformCardStats(rawStats)

	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestTransformAllStats_AllIncluded(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	rawStats := json.RawMessage(`[
		{"code":"ac%","value":163,"displayText":"+163% Enhanced Defense","isVariable":true},
		{"code":"str","value":2,"displayText":"+2 to Strength","isVariable":false},
		{"code":"dr","value":10,"displayText":"Damage Reduced by 10%","isVariable":true}
	]`)

	result := svc.transformAllStats(rawStats)

	assert.Len(t, result, 3)

	// First stat: variable
	assert.Equal(t, "ac%", result[0].Code)
	assert.Equal(t, intPtr(163), result[0].Value)
	assert.Equal(t, "+163% Enhanced Defense", result[0].DisplayText)
	assert.True(t, result[0].IsVariable)

	// Second stat: not variable
	assert.Equal(t, "str", result[1].Code)
	assert.Equal(t, intPtr(2), result[1].Value)
	assert.False(t, result[1].IsVariable)

	// Third stat: variable
	assert.Equal(t, "dr", result[2].Code)
	assert.True(t, result[2].IsVariable)
}

func TestTransformAllStats_DisplayTextFallback(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	rawStats := json.RawMessage(`[
		{"code":"ac%","value":100,"displayText":""},
		{"code":"mana","displayText":""}
	]`)

	result := svc.transformAllStats(rawStats)

	assert.Len(t, result, 2)
	// With value: fallback to "code: value"
	assert.Equal(t, "ac%: 100", result[0].DisplayText)
	// Without numeric value: fallback to just code
	assert.Equal(t, "mana", result[1].DisplayText)
}

func TestTransformAllStats_StringValue(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	rawStats := json.RawMessage(`[
		{"code":"ed%","value":"+163% Enhanced Defense","displayText":"+163% Enhanced Defense","isVariable":true}
	]`)

	result := svc.transformAllStats(rawStats)

	assert.Len(t, result, 1)
	assert.Equal(t, "ed%", result[0].Code)
	// extractNumericValue should extract 163 from the string
	assert.NotNil(t, result[0].Value)
	assert.Equal(t, 163, *result[0].Value)
}

func TestTransformAllStats_NilValue(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	rawStats := json.RawMessage(`[
		{"code":"skill","displayText":"All Skills","isVariable":false}
	]`)

	result := svc.transformAllStats(rawStats)

	assert.Len(t, result, 1)
	assert.Equal(t, "skill", result[0].Code)
	assert.Nil(t, result[0].Value)
}

func TestExtractNumericValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected *int
	}{
		{
			name:     "float64 value",
			input:    float64(163),
			expected: intPtr(163),
		},
		{
			name:     "int value",
			input:    int(42),
			expected: intPtr(42),
		},
		{
			name:     "string with number",
			input:    "+163% Enhanced Defense",
			expected: intPtr(163),
		},
		{
			name:     "string with negative number",
			input:    "-25 to Enemy Fire Resistance",
			expected: intPtr(-25),
		},
		{
			name:     "nil value",
			input:    nil,
			expected: nil,
		},
		{
			name:     "string without number",
			input:    "All Skills",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractNumericValue(tt.input)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Rune Transformation
// ---------------------------------------------------------------------------

func TestTransformRunes_ValidCodes(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	rawRunes := json.RawMessage(`["r31","r06","r30"]`) // Jah, Ith, Ber

	result := svc.transformRunes(rawRunes)

	assert.Len(t, result, 3)

	assert.Equal(t, "r31", result[0].Code)
	assert.Equal(t, "Jah", result[0].Name)
	assert.Contains(t, result[0].ImageURL, "jah.png")

	assert.Equal(t, "r06", result[1].Code)
	assert.Equal(t, "Ith", result[1].Name)
	assert.Contains(t, result[1].ImageURL, "ith.png")

	assert.Equal(t, "r30", result[2].Code)
	assert.Equal(t, "Ber", result[2].Name)
	assert.Contains(t, result[2].ImageURL, "ber.png")
}

func TestTransformRunes_Empty(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	// nil
	result := svc.transformRunes(nil)
	assert.Nil(t, result)

	// empty
	result2 := svc.transformRunes(json.RawMessage{})
	assert.Nil(t, result2)
}

func TestTransformRunes_InvalidJSON(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	result := svc.transformRunes(json.RawMessage(`not valid json`))
	assert.Nil(t, result)
}

// ---------------------------------------------------------------------------
// DTO Mapping
// ---------------------------------------------------------------------------

func TestToCardResponse_WithSeller(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	seller := testProfile(testSellerID, withPremium)
	listing := testListing(testListingID, testSellerID, withSeller(seller))

	resp := svc.ToCardResponse(listing)

	assert.Equal(t, testListingID, resp.ID)
	assert.Equal(t, testSellerID, resp.SellerID)
	assert.Equal(t, "Shako", resp.Name)
	assert.NotNil(t, resp.Seller)
	assert.Equal(t, testSellerID, resp.Seller.ID)
	assert.True(t, resp.Seller.IsPremium)
}

func TestToCardResponse_NoSeller(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	listing := testListing(testListingID, testSellerID) // no seller set

	resp := svc.ToCardResponse(listing)

	assert.Equal(t, testListingID, resp.ID)
	assert.Nil(t, resp.Seller)
}

func TestToResponse_AllFields(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	rawStats := json.RawMessage(`[{"code":"ac%","value":163,"displayText":"+163% Enhanced Defense","isVariable":true}]`)
	rawRunes := json.RawMessage(`["r31","r06","r30"]`)
	seller := testProfile(testSellerID)

	listing := testListing(testListingID, testSellerID,
		withStats(rawStats),
		withRunes(rawRunes),
		withSeller(seller),
	)
	listing.ImageURL = strPtr("https://example.com/image.png")
	listing.AskingPrice = strPtr("Ber rune")
	listing.Notes = strPtr("Perfect roll")
	listing.RuneOrder = strPtr("JahIthBer")
	listing.BaseItemCode = strPtr("uap")
	listing.BaseItemName = strPtr("Shako")

	resp := svc.ToResponse(listing)

	assert.Equal(t, testListingID, resp.ID)
	assert.Equal(t, testSellerID, resp.SellerID)
	assert.Equal(t, "Shako", resp.Name)
	assert.Equal(t, "unique", resp.ItemType)
	assert.Equal(t, "unique", resp.Rarity)
	assert.Equal(t, "https://example.com/image.png", resp.ImageURL)
	assert.Equal(t, "helm", resp.Category)
	assert.Equal(t, "Ber rune", resp.AskingPrice)
	assert.Equal(t, "Perfect roll", resp.Notes)
	assert.Equal(t, "JahIthBer", resp.RuneOrder)
	assert.Equal(t, "uap", resp.BaseItemCode)
	assert.Equal(t, "Shako", resp.BaseItemName)
	assert.Equal(t, "active", resp.Status)

	// Stats should include all stats (transformAllStats)
	assert.Len(t, resp.Stats, 1)
	assert.Equal(t, "ac%", resp.Stats[0].Code)
	assert.True(t, resp.Stats[0].IsVariable)

	// Runes should be transformed
	assert.Len(t, resp.Runes, 3)
	assert.Equal(t, "Jah", resp.Runes[0].Name)

	// Seller mapped
	assert.NotNil(t, resp.Seller)
	assert.Equal(t, testSellerID, resp.Seller.ID)
}

// ---------------------------------------------------------------------------
// Filter Parsing (List)
// ---------------------------------------------------------------------------

func TestList_ParsesAffixFilters(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	listingRepo.On("List", mock.Anything, mock.MatchedBy(func(f repository.ListingFilter) bool {
		if len(f.AffixFilters) != 1 {
			return false
		}
		af := f.AffixFilters[0]
		return af.Code == "ed%" && af.MinValue != nil && *af.MinValue == 150 && af.MaxValue == nil
	})).Return([]*models.Listing{}, 0, nil)

	req := &dto.ListingFilterRequest{
		AffixFilters: `[{"code":"ed%","minValue":150}]`,
	}

	listings, count, err := svc.List(context.Background(), req)

	assert.NoError(t, err)
	assert.Empty(t, listings)
	assert.Equal(t, 0, count)
	listingRepo.AssertExpectations(t)
}

func TestList_ParsesAskingForFilter(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	listingRepo.On("List", mock.Anything, mock.MatchedBy(func(f repository.ListingFilter) bool {
		return f.AskingForFilter != nil &&
			f.AskingForFilter.Name == "Ber" &&
			f.AskingForFilter.Type == "rune"
	})).Return([]*models.Listing{}, 0, nil)

	req := &dto.ListingFilterRequest{
		AskingForFilters: `{"name":"Ber","type":"rune"}`,
	}

	_, _, err := svc.List(context.Background(), req)

	assert.NoError(t, err)
	listingRepo.AssertExpectations(t)
}

func TestList_EmptyFilters(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	listingRepo.On("List", mock.Anything, mock.MatchedBy(func(f repository.ListingFilter) bool {
		return f.Query == "" &&
			f.Game == "" &&
			f.Rarity == "" &&
			len(f.AffixFilters) == 0 &&
			f.AskingForFilter == nil &&
			len(f.Platforms) == 0 &&
			len(f.Categories) == 0
	})).Return([]*models.Listing{}, 0, nil)

	req := &dto.ListingFilterRequest{
		Pagination: dto.Pagination{Page: 1, PerPage: 20},
	}

	listings, count, err := svc.List(context.Background(), req)

	assert.NoError(t, err)
	assert.Empty(t, listings)
	assert.Equal(t, 0, count)
	listingRepo.AssertExpectations(t)
}

func TestList_ParsesMultipleGameModeFilters(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	ladder := true
	hardcore := false
	isNonRotw := true

	listingRepo.On("List", mock.Anything, mock.MatchedBy(func(f repository.ListingFilter) bool {
		return f.Game == "diablo2" &&
			f.Ladder != nil && *f.Ladder == true &&
			f.Hardcore != nil && *f.Hardcore == false &&
			f.IsNonRotw != nil && *f.IsNonRotw == true
	})).Return([]*models.Listing{}, 0, nil)

	req := &dto.ListingFilterRequest{
		Game:      "diablo2",
		Ladder:    &ladder,
		Hardcore:  &hardcore,
		IsNonRotw: &isNonRotw,
	}

	_, _, err := svc.List(context.Background(), req)

	assert.NoError(t, err)
	listingRepo.AssertExpectations(t)
}

func TestList_ParsesPlatformFilter(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	listingRepo.On("List", mock.Anything, mock.MatchedBy(func(f repository.ListingFilter) bool {
		return len(f.Platforms) == 2 &&
			f.Platforms[0] == "pc" &&
			f.Platforms[1] == "xbox"
	})).Return([]*models.Listing{}, 0, nil)

	req := &dto.ListingFilterRequest{
		Platforms: "pc,xbox",
	}

	_, _, err := svc.List(context.Background(), req)

	assert.NoError(t, err)
	listingRepo.AssertExpectations(t)
}

func TestList_ParsesCategories(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	listingRepo.On("List", mock.Anything, mock.MatchedBy(func(f repository.ListingFilter) bool {
		return len(f.Categories) == 3 &&
			f.Categories[0] == "helms" &&
			f.Categories[1] == "body armor" &&
			f.Categories[2] == "weapons"
	})).Return([]*models.Listing{}, 0, nil)

	req := &dto.ListingFilterRequest{
		Categories: "helms,body armor,weapons",
	}

	_, _, err := svc.List(context.Background(), req)

	assert.NoError(t, err)
	listingRepo.AssertExpectations(t)
}

func TestList_ParsesRarity(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	listingRepo.On("List", mock.Anything, mock.MatchedBy(func(f repository.ListingFilter) bool {
		return f.Rarity == "unique"
	})).Return([]*models.Listing{}, 0, nil)

	req := &dto.ListingFilterRequest{
		Rarity: "unique",
	}

	_, _, err := svc.List(context.Background(), req)

	assert.NoError(t, err)
	listingRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// parsePlatforms
// ---------------------------------------------------------------------------

func TestParsePlatforms(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "comma separated",
			input:    "pc,xbox,playstation",
			expected: []string{"pc", "xbox", "playstation"},
		},
		{
			name:     "with spaces",
			input:    "pc, xbox, playstation",
			expected: []string{"pc", "xbox", "playstation"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single value",
			input:    "pc",
			expected: []string{"pc"},
		},
		{
			name:     "trailing comma ignored",
			input:    "pc,xbox,",
			expected: []string{"pc", "xbox"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePlatforms(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// IncrementViews
// ---------------------------------------------------------------------------

func TestIncrementViews_Success(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	listingRepo := new(mocks.MockListingRepository)
	svc, _ := setupListingService(profileRepo, listingRepo, newTestRedis())

	listingRepo.On("IncrementViews", mock.Anything, testListingID).Return(nil)

	err := svc.IncrementViews(context.Background(), testListingID)

	assert.NoError(t, err)
	listingRepo.AssertExpectations(t)
}
