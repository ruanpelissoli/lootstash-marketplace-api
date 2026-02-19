package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---------- helpers ----------

func newWishlistTestService() (
	*WishlistService,
	*mocks.MockWishlistRepository,
	*mocks.MockProfileRepository,
	*mocks.MockNotificationRepository,
) {
	wishlistRepo := new(mocks.MockWishlistRepository)
	profileRepo := new(mocks.MockProfileRepository)
	notifRepo := new(mocks.MockNotificationRepository)

	profileService := NewProfileService(profileRepo, nil, nil)
	notifService := NewNotificationService(notifRepo, nil)
	svc := NewWishlistService(wishlistRepo, profileService, notifService)

	return svc, wishlistRepo, profileRepo, notifRepo
}

// ---------- Create ----------

func TestWishlistCreate_PremiumUser_Success(t *testing.T) {
	svc, wishlistRepo, profileRepo, _ := newWishlistTestService()
	ctx := context.Background()

	profile := testProfile(testUserID, withPremium)
	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil)
	wishlistRepo.On("CountActiveByUserID", ctx, testUserID).Return(5, nil)
	wishlistRepo.On("Create", ctx, mock.AnythingOfType("*models.WishlistItem")).Return(nil)

	req := &dto.CreateWishlistItemRequest{
		Name:     "Shako",
		Category: strPtr("helms"),
		Rarity:   strPtr("unique"),
		Game:     "diablo2",
	}

	item, err := svc.Create(ctx, testUserID, req)

	require.NoError(t, err)
	assert.Equal(t, testUserID, item.UserID)
	assert.Equal(t, "Shako", item.Name)
	assert.Equal(t, "active", item.Status)
	assert.NotEmpty(t, item.ID)
	wishlistRepo.AssertCalled(t, "Create", ctx, mock.AnythingOfType("*models.WishlistItem"))
}

func TestWishlistCreate_FreeUser(t *testing.T) {
	svc, _, profileRepo, _ := newWishlistTestService()
	ctx := context.Background()

	profile := testProfile(testUserID) // not premium
	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil)

	req := &dto.CreateWishlistItemRequest{
		Name: "Shako",
		Game: "diablo2",
	}

	item, err := svc.Create(ctx, testUserID, req)

	assert.Nil(t, item)
	assert.ErrorIs(t, err, ErrPremiumRequired)
}

func TestWishlistCreate_AtLimit(t *testing.T) {
	svc, wishlistRepo, profileRepo, _ := newWishlistTestService()
	ctx := context.Background()

	profile := testProfile(testUserID, withPremium)
	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil)
	wishlistRepo.On("CountActiveByUserID", ctx, testUserID).Return(10, nil)

	req := &dto.CreateWishlistItemRequest{
		Name: "Shako",
		Game: "diablo2",
	}

	item, err := svc.Create(ctx, testUserID, req)

	assert.Nil(t, item)
	assert.ErrorIs(t, err, ErrWishlistLimitReached)
}

func TestWishlistCreate_ConvertsCriteria(t *testing.T) {
	svc, wishlistRepo, profileRepo, _ := newWishlistTestService()
	ctx := context.Background()

	profile := testProfile(testUserID, withPremium)
	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil)
	wishlistRepo.On("CountActiveByUserID", ctx, testUserID).Return(0, nil)

	var capturedItem *models.WishlistItem
	wishlistRepo.On("Create", ctx, mock.AnythingOfType("*models.WishlistItem")).
		Run(func(args mock.Arguments) {
			capturedItem = args.Get(1).(*models.WishlistItem)
		}).
		Return(nil)

	req := &dto.CreateWishlistItemRequest{
		Name: "Shako",
		Game: "diablo2",
		StatCriteria: []dto.StatCriterionDTO{
			{Code: "ed%", Name: "Enhanced Defense", MinValue: intPtr(100), MaxValue: intPtr(200)},
			{Code: "ac%", Name: "Enhanced Armor", MinValue: intPtr(50)},
		},
	}

	item, err := svc.Create(ctx, testUserID, req)

	require.NoError(t, err)
	require.NotNil(t, item)
	require.NotNil(t, capturedItem)
	require.Len(t, capturedItem.StatCriteria, 2)

	assert.Equal(t, "ed%", capturedItem.StatCriteria[0].Code)
	assert.Equal(t, "Enhanced Defense", capturedItem.StatCriteria[0].Name)
	assert.Equal(t, intPtr(100), capturedItem.StatCriteria[0].MinValue)
	assert.Equal(t, intPtr(200), capturedItem.StatCriteria[0].MaxValue)

	assert.Equal(t, "ac%", capturedItem.StatCriteria[1].Code)
	assert.Equal(t, intPtr(50), capturedItem.StatCriteria[1].MinValue)
	assert.Nil(t, capturedItem.StatCriteria[1].MaxValue)
}

// ---------- List ----------

func TestWishlistList_PremiumUser_Success(t *testing.T) {
	svc, wishlistRepo, profileRepo, _ := newWishlistTestService()
	ctx := context.Background()

	profile := testProfile(testUserID, withPremium)
	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil)

	items := []*models.WishlistItem{
		testWishlistItem("wl-1", testUserID),
		testWishlistItem("wl-2", testUserID),
	}
	wishlistRepo.On("ListByUserID", ctx, testUserID, 0, 20).Return(items, 2, nil)

	result, total, err := svc.List(ctx, testUserID, 0, 20)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 2, total)
}

func TestWishlistList_FreeUser(t *testing.T) {
	svc, _, profileRepo, _ := newWishlistTestService()
	ctx := context.Background()

	profile := testProfile(testUserID) // not premium
	profileRepo.On("GetByID", ctx, testUserID).Return(profile, nil)

	result, total, err := svc.List(ctx, testUserID, 0, 20)

	assert.Nil(t, result)
	assert.Equal(t, 0, total)
	assert.ErrorIs(t, err, ErrPremiumRequired)
}

// ---------- Update ----------

func TestWishlistUpdate_Success(t *testing.T) {
	svc, wishlistRepo, _, _ := newWishlistTestService()
	ctx := context.Background()

	existing := testWishlistItem(testWishlistID, testUserID)
	wishlistRepo.On("GetByID", ctx, testWishlistID).Return(existing, nil)
	wishlistRepo.On("Update", ctx, mock.AnythingOfType("*models.WishlistItem")).Return(nil)

	req := &dto.UpdateWishlistItemRequest{
		Name:   strPtr("Updated Shako"),
		Status: strPtr("paused"),
	}

	item, err := svc.Update(ctx, testWishlistID, testUserID, req)

	require.NoError(t, err)
	assert.Equal(t, "Updated Shako", item.Name)
	assert.Equal(t, "paused", item.Status)
	wishlistRepo.AssertCalled(t, "Update", ctx, mock.AnythingOfType("*models.WishlistItem"))
}

func TestWishlistUpdate_NotOwner(t *testing.T) {
	svc, wishlistRepo, _, _ := newWishlistTestService()
	ctx := context.Background()

	existing := testWishlistItem(testWishlistID, testUserID)
	wishlistRepo.On("GetByID", ctx, testWishlistID).Return(existing, nil)

	req := &dto.UpdateWishlistItemRequest{
		Name: strPtr("Hacked"),
	}

	item, err := svc.Update(ctx, testWishlistID, "other-user-id", req)

	assert.Nil(t, item)
	assert.ErrorIs(t, err, ErrForbidden)
}

// ---------- Delete ----------

func TestWishlistDelete_Success(t *testing.T) {
	svc, wishlistRepo, _, _ := newWishlistTestService()
	ctx := context.Background()

	existing := testWishlistItem(testWishlistID, testUserID)
	wishlistRepo.On("GetByID", ctx, testWishlistID).Return(existing, nil)
	wishlistRepo.On("Delete", ctx, testWishlistID).Return(nil)

	err := svc.Delete(ctx, testWishlistID, testUserID)

	require.NoError(t, err)
	wishlistRepo.AssertCalled(t, "Delete", ctx, testWishlistID)
}

func TestWishlistDelete_NotOwner(t *testing.T) {
	svc, wishlistRepo, _, _ := newWishlistTestService()
	ctx := context.Background()

	existing := testWishlistItem(testWishlistID, testUserID)
	wishlistRepo.On("GetByID", ctx, testWishlistID).Return(existing, nil)

	err := svc.Delete(ctx, testWishlistID, "other-user-id")

	assert.ErrorIs(t, err, ErrForbidden)
	wishlistRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

// ---------- CheckAndNotifyMatches ----------

func makeListingWithStats() *models.Listing {
	stats, _ := json.Marshal([]map[string]interface{}{
		{"code": "ed%", "value": 163.0, "isVariable": true},
		{"code": "ac%", "value": 100.0, "isVariable": true},
	})
	return testListing(testListingID, testSellerID, withStats(stats))
}

func TestCheckAndNotifyMatches_NoCandidates(t *testing.T) {
	svc, wishlistRepo, _, notifRepo := newWishlistTestService()
	ctx := context.Background()

	listing := makeListingWithStats()
	wishlistRepo.On("FindMatchingItems", ctx, listing).Return([]*models.WishlistItem{}, nil)

	svc.CheckAndNotifyMatches(ctx, listing)

	wishlistRepo.AssertCalled(t, "FindMatchingItems", ctx, listing)
	notifRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestCheckAndNotifyMatches_MatchesAllCriteria(t *testing.T) {
	svc, wishlistRepo, _, notifRepo := newWishlistTestService()
	ctx := context.Background()

	listing := makeListingWithStats()

	candidate := testWishlistItem("wl-1", "user-abc", withStatCriteria([]models.StatCriterion{
		{Code: "ed%", MinValue: intPtr(150), MaxValue: intPtr(200)},
	}))
	wishlistRepo.On("FindMatchingItems", ctx, listing).Return([]*models.WishlistItem{candidate}, nil)
	notifRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Notification")).Return(nil)

	svc.CheckAndNotifyMatches(ctx, listing)

	notifRepo.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("*models.Notification"))
}

func TestCheckAndNotifyMatches_FailsMinValue(t *testing.T) {
	svc, wishlistRepo, _, notifRepo := newWishlistTestService()
	ctx := context.Background()

	// Listing has ed%=163. Candidate requires min 200 -> should NOT match.
	listing := makeListingWithStats()

	candidate := testWishlistItem("wl-1", "user-abc", withStatCriteria([]models.StatCriterion{
		{Code: "ed%", MinValue: intPtr(200)},
	}))
	wishlistRepo.On("FindMatchingItems", ctx, listing).Return([]*models.WishlistItem{candidate}, nil)

	svc.CheckAndNotifyMatches(ctx, listing)

	notifRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestCheckAndNotifyMatches_FailsMaxValue(t *testing.T) {
	svc, wishlistRepo, _, notifRepo := newWishlistTestService()
	ctx := context.Background()

	// Listing has ed%=163. Candidate requires max 150 -> should NOT match.
	listing := makeListingWithStats()

	candidate := testWishlistItem("wl-1", "user-abc", withStatCriteria([]models.StatCriterion{
		{Code: "ed%", MaxValue: intPtr(150)},
	}))
	wishlistRepo.On("FindMatchingItems", ctx, listing).Return([]*models.WishlistItem{candidate}, nil)

	svc.CheckAndNotifyMatches(ctx, listing)

	notifRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestCheckAndNotifyMatches_NoCriteria_AlwaysMatches(t *testing.T) {
	svc, wishlistRepo, _, notifRepo := newWishlistTestService()
	ctx := context.Background()

	listing := makeListingWithStats()

	// Candidate with empty stat criteria should always match.
	candidate := testWishlistItem("wl-1", "user-abc")
	candidate.StatCriteria = nil
	wishlistRepo.On("FindMatchingItems", ctx, listing).Return([]*models.WishlistItem{candidate}, nil)
	notifRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Notification")).Return(nil)

	svc.CheckAndNotifyMatches(ctx, listing)

	notifRepo.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("*models.Notification"))
}

func TestCheckAndNotifyMatches_MissingStatCode_NoMatch(t *testing.T) {
	svc, wishlistRepo, _, notifRepo := newWishlistTestService()
	ctx := context.Background()

	listing := makeListingWithStats()

	// Candidate requires "fcr%" which is not in listing stats.
	candidate := testWishlistItem("wl-1", "user-abc", withStatCriteria([]models.StatCriterion{
		{Code: "fcr%", MinValue: intPtr(10)},
	}))
	wishlistRepo.On("FindMatchingItems", ctx, listing).Return([]*models.WishlistItem{candidate}, nil)

	svc.CheckAndNotifyMatches(ctx, listing)

	notifRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestCheckAndNotifyMatches_MultipleMatches(t *testing.T) {
	svc, wishlistRepo, _, notifRepo := newWishlistTestService()
	ctx := context.Background()

	listing := makeListingWithStats()

	candidate1 := testWishlistItem("wl-1", "user-abc", withStatCriteria([]models.StatCriterion{
		{Code: "ed%", MinValue: intPtr(150), MaxValue: intPtr(200)},
	}))
	candidate2 := testWishlistItem("wl-2", "user-xyz", withStatCriteria([]models.StatCriterion{
		{Code: "ac%", MinValue: intPtr(50), MaxValue: intPtr(150)},
	}))

	wishlistRepo.On("FindMatchingItems", ctx, listing).Return([]*models.WishlistItem{candidate1, candidate2}, nil)
	notifRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Notification")).Return(nil)

	svc.CheckAndNotifyMatches(ctx, listing)

	// Both candidates should trigger notifications.
	assert.Equal(t, 2, len(notifRepo.Calls))
}

// ---------- matchesStatCriteria (direct, same package) ----------

func TestMatchesStatCriteria_EmptyCriteria(t *testing.T) {
	svc := &WishlistService{}
	statMap := map[string]int{"ed%": 163, "ac%": 100}

	result := svc.matchesStatCriteria(nil, statMap, slog.Default())

	assert.True(t, result)
}

func TestMatchesStatCriteria_AllPass(t *testing.T) {
	svc := &WishlistService{}
	statMap := map[string]int{"ed%": 163, "ac%": 100}
	criteria := []models.StatCriterion{
		{Code: "ed%", MinValue: intPtr(150), MaxValue: intPtr(200)},
		{Code: "ac%", MinValue: intPtr(50), MaxValue: intPtr(150)},
	}

	result := svc.matchesStatCriteria(criteria, statMap, slog.Default())

	assert.True(t, result)
}

func TestMatchesStatCriteria_BelowMin(t *testing.T) {
	svc := &WishlistService{}
	statMap := map[string]int{"ed%": 100}
	criteria := []models.StatCriterion{
		{Code: "ed%", MinValue: intPtr(150)},
	}

	result := svc.matchesStatCriteria(criteria, statMap, slog.Default())

	assert.False(t, result)
}

func TestMatchesStatCriteria_AboveMax(t *testing.T) {
	svc := &WishlistService{}
	statMap := map[string]int{"ed%": 200}
	criteria := []models.StatCriterion{
		{Code: "ed%", MaxValue: intPtr(150)},
	}

	result := svc.matchesStatCriteria(criteria, statMap, slog.Default())

	assert.False(t, result)
}

func TestMatchesStatCriteria_CodeNotFound(t *testing.T) {
	svc := &WishlistService{}
	statMap := map[string]int{"ed%": 163}
	criteria := []models.StatCriterion{
		{Code: "fcr%", MinValue: intPtr(10)},
	}

	result := svc.matchesStatCriteria(criteria, statMap, slog.Default())

	assert.False(t, result)
}
