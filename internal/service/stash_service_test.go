package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

type stashTestEnv struct {
	svc            *StashService
	stashRepo      *mocks.MockStashItemRepository
	listingRepo    *mocks.MockListingRepository
	profileRepo    *mocks.MockProfileRepository
	listingService *ListingService
}

func newStashTestEnv() *stashTestEnv {
	stashRepo := new(mocks.MockStashItemRepository)
	listingRepo := new(mocks.MockListingRepository)
	profileRepo := new(mocks.MockProfileRepository)

	profileService := NewProfileService(profileRepo, nil, nil)
	listingService := NewListingService(listingRepo, profileService, nil)

	// Set stash repo on listing service so auto-stash creation logic is active
	listingService.SetStashRepository(stashRepo)

	svc := NewStashService(stashRepo, listingRepo, profileService, nil, nil, "", "")
	svc.SetListingService(listingService)

	return &stashTestEnv{
		svc:            svc,
		stashRepo:      stashRepo,
		listingRepo:    listingRepo,
		profileRepo:    profileRepo,
		listingService: listingService,
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestStashCreate_ManualItem_Success(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	env.stashRepo.On("Create", ctx, mock.AnythingOfType("*models.StashItem")).Return(nil)

	req := &dto.CreateStashItemRequest{
		Name:     "Harlequin Crest",
		ItemType: "Shako",
		Rarity:   "unique",
		Category: "helms",
		Game:     "diablo2",
		Platforms: []string{"pc"},
		Region:   "americas",
	}

	item, err := env.svc.Create(ctx, testUserID, req)

	assert.NoError(t, err)
	assert.NotNil(t, item)
	assert.Equal(t, testUserID, item.UserID)
	assert.Equal(t, "Shako", item.ItemType)
	assert.Equal(t, "Harlequin Crest", *item.Name)
	assert.Equal(t, "manual", item.Source)
	assert.Equal(t, 1, item.Quantity)
	env.stashRepo.AssertExpectations(t)
}

func TestStashCreate_StackableItem_QuantityHonored(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	env.stashRepo.On("Create", ctx, mock.AnythingOfType("*models.StashItem")).Return(nil)

	qty := 5
	req := &dto.CreateStashItemRequest{
		ItemType:  "Amethyst",
		Rarity:    "normal",
		Category:  "gem",
		Game:      "diablo2",
		Quantity:  &qty,
		Platforms: []string{"pc"},
		Region:    "americas",
	}

	item, err := env.svc.Create(ctx, testUserID, req)

	assert.NoError(t, err)
	assert.Equal(t, 5, item.Quantity)
	assert.True(t, item.IsStackable())
	env.stashRepo.AssertExpectations(t)
}

func TestStashCreate_NonStackableItem_QuantityForcedTo1(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	env.stashRepo.On("Create", ctx, mock.AnythingOfType("*models.StashItem")).Return(nil)

	qty := 10
	req := &dto.CreateStashItemRequest{
		ItemType:  "Shako",
		Rarity:    "unique",
		Category:  "helms",
		Game:      "diablo2",
		Quantity:  &qty,
		Platforms: []string{"pc"},
		Region:    "americas",
	}

	item, err := env.svc.Create(ctx, testUserID, req)

	assert.NoError(t, err)
	assert.Equal(t, 1, item.Quantity) // forced to 1 because helms are not stackable
	env.stashRepo.AssertExpectations(t)
}

func TestStashCreate_NoName_NameIsNil(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	env.stashRepo.On("Create", ctx, mock.AnythingOfType("*models.StashItem")).Return(nil)

	req := &dto.CreateStashItemRequest{
		ItemType:  "Ring",
		Rarity:    "rare",
		Category:  "rings",
		Game:      "diablo2",
		Platforms: []string{"pc"},
		Region:    "americas",
	}

	item, err := env.svc.Create(ctx, testUserID, req)

	assert.NoError(t, err)
	assert.Nil(t, item.Name)
	assert.Equal(t, "Ring", item.GetDisplayName())
	env.stashRepo.AssertExpectations(t)
}

func TestStashCreate_RepoError(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	env.stashRepo.On("Create", ctx, mock.AnythingOfType("*models.StashItem")).Return(errors.New("db error"))

	req := &dto.CreateStashItemRequest{
		ItemType:  "Shako",
		Rarity:    "unique",
		Category:  "helms",
		Game:      "diablo2",
		Platforms: []string{"pc"},
		Region:    "americas",
	}

	item, err := env.svc.Create(ctx, testUserID, req)

	assert.Error(t, err)
	assert.Nil(t, item)
	env.stashRepo.AssertExpectations(t)
}

func TestStashCreate_OptionalFields(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	env.stashRepo.On("Create", ctx, mock.AnythingOfType("*models.StashItem")).Return(nil)

	req := &dto.CreateStashItemRequest{
		Name:          "Insight",
		ItemType:      "Polearm",
		Rarity:        "runeword",
		Category:      "weapons",
		ImageURL:      "https://example.com/insight.png",
		RuneOrder:     "r08r03r07r12",
		BaseItemCode:  "9vo",
		BaseItemName:  "Voulge",
		CatalogItemID: "insight",
		Game:          "diablo2",
		Platforms:     []string{"pc"},
		Region:        "americas",
	}

	item, err := env.svc.Create(ctx, testUserID, req)

	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/insight.png", *item.ImageURL)
	assert.Equal(t, "r08r03r07r12", *item.RuneOrder)
	assert.Equal(t, "9vo", *item.BaseItemCode)
	assert.Equal(t, "Voulge", *item.BaseItemName)
	assert.Equal(t, "insight", *item.CatalogItemID)
	env.stashRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestStashGetByID_Success(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID, withStashName("Shako"))
	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(item, nil)

	result, err := env.svc.GetByID(ctx, testStashItemID, testUserID)

	assert.NoError(t, err)
	assert.Equal(t, testStashItemID, result.ID)
	env.stashRepo.AssertExpectations(t)
}

func TestStashGetByID_NotFound(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(nil, sql.ErrNoRows)

	result, err := env.svc.GetByID(ctx, testStashItemID, testUserID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, sql.ErrNoRows)
	env.stashRepo.AssertExpectations(t)
}

func TestStashGetByID_WrongUser(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testBuyerID).Return(nil, sql.ErrNoRows)

	result, err := env.svc.GetByID(ctx, testStashItemID, testBuyerID)

	assert.Error(t, err)
	assert.Nil(t, result)
	env.stashRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestStashDelete_Success(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID)
	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(item, nil)
	env.listingRepo.On("GetActiveListingByStashItemID", ctx, testStashItemID).Return(nil, sql.ErrNoRows)
	env.stashRepo.On("Delete", ctx, testStashItemID).Return(nil)

	err := env.svc.Delete(ctx, testStashItemID, testUserID, nil)

	assert.NoError(t, err)
	env.stashRepo.AssertExpectations(t)
	env.listingRepo.AssertExpectations(t)
}

func TestStashDelete_CancelsLinkedListing(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID)
	listing := testListing("listing-1", testUserID, withStashStashItemID(&item.ID))

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(item, nil)
	env.listingRepo.On("GetActiveListingByStashItemID", ctx, testStashItemID).Return(listing, nil)

	// listingService.Delete() calls GetByID then Update
	env.listingRepo.On("GetByID", ctx, "listing-1").Return(listing, nil)
	env.listingRepo.On("Update", ctx, mock.MatchedBy(func(l *models.Listing) bool {
		return l.ID == "listing-1" && l.Status == "cancelled"
	})).Return(nil)

	env.stashRepo.On("Delete", ctx, testStashItemID).Return(nil)

	err := env.svc.Delete(ctx, testStashItemID, testUserID, nil)

	assert.NoError(t, err)
	env.stashRepo.AssertExpectations(t)
	env.listingRepo.AssertExpectations(t)
}

func TestStashDelete_LinkedListingsFetchError_StillDeletes(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID)
	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(item, nil)
	env.listingRepo.On("GetActiveListingByStashItemID", ctx, testStashItemID).Return(nil, errors.New("db error"))
	env.stashRepo.On("Delete", ctx, testStashItemID).Return(nil)

	err := env.svc.Delete(ctx, testStashItemID, testUserID, nil)

	assert.NoError(t, err) // stash item still deleted despite listing fetch error
	env.stashRepo.AssertExpectations(t)
}

func TestStashDelete_NotFound(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(nil, sql.ErrNoRows)

	err := env.svc.Delete(ctx, testStashItemID, testUserID, nil)

	assert.Error(t, err)
	assert.ErrorIs(t, err, sql.ErrNoRows)
	env.stashRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestStashDelete_WrongUser(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testBuyerID).Return(nil, sql.ErrNoRows)

	err := env.svc.Delete(ctx, testStashItemID, testBuyerID, nil)

	assert.Error(t, err)
	env.stashRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestStashDelete_RepoError(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID)
	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(item, nil)
	env.listingRepo.On("GetActiveListingByStashItemID", ctx, testStashItemID).Return(nil, sql.ErrNoRows)
	env.stashRepo.On("Delete", ctx, testStashItemID).Return(errors.New("db error"))

	err := env.svc.Delete(ctx, testStashItemID, testUserID, nil)

	assert.Error(t, err)
	env.stashRepo.AssertExpectations(t)
}

func TestStashDelete_ListedOnly_PartiallyListed_KeepsUnlisted(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	// Rune x5, with 3 listed
	item := testStashItem(testStashItemID, testUserID, withStashCategory("rune"), withStashQuantity(5))
	listing := testListing("listing-1", testUserID, withStashStashItemID(&item.ID), withListingAmount(3))

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(item, nil)
	env.listingRepo.On("GetActiveListingByStashItemID", ctx, testStashItemID).Return(listing, nil)

	// Listing gets cancelled
	env.listingRepo.On("GetByID", ctx, "listing-1").Return(listing, nil)
	env.listingRepo.On("Update", ctx, mock.MatchedBy(func(l *models.Listing) bool {
		return l.ID == "listing-1" && l.Status == "cancelled"
	})).Return(nil)

	// Quantity decremented by listed amount (3), NOT hard-deleted
	env.stashRepo.On("DecrementQuantity", ctx, testStashItemID, 3).Return(nil)

	err := env.svc.Delete(ctx, testStashItemID, testUserID, boolPtr(true))

	assert.NoError(t, err)
	// Item should NOT be hard-deleted — unlisted qty (2) remains
	env.stashRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	env.stashRepo.AssertExpectations(t)
	env.listingRepo.AssertExpectations(t)
}

func TestStashDelete_ListedOnly_AllListed_DeletesItem(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	// Rune x3, all 3 listed
	item := testStashItem(testStashItemID, testUserID, withStashCategory("rune"), withStashQuantity(3))
	listing := testListing("listing-1", testUserID, withStashStashItemID(&item.ID), withListingAmount(3))

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(item, nil)
	env.listingRepo.On("GetActiveListingByStashItemID", ctx, testStashItemID).Return(listing, nil)

	env.listingRepo.On("GetByID", ctx, "listing-1").Return(listing, nil)
	env.listingRepo.On("Update", ctx, mock.MatchedBy(func(l *models.Listing) bool {
		return l.ID == "listing-1" && l.Status == "cancelled"
	})).Return(nil)

	// All listed = 0 unlisted → full delete
	env.stashRepo.On("Delete", ctx, testStashItemID).Return(nil)

	err := env.svc.Delete(ctx, testStashItemID, testUserID, boolPtr(true))

	assert.NoError(t, err)
	env.stashRepo.AssertExpectations(t)
	env.listingRepo.AssertExpectations(t)
}

func TestStashDelete_UnlistedOnly_PartiallyListed_KeepsListed(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	// Rune x5, with 3 listed → 2 unlisted
	item := testStashItem(testStashItemID, testUserID, withStashCategory("rune"), withStashQuantity(5))
	listing := testListing("listing-1", testUserID, withStashStashItemID(&item.ID), withListingAmount(3))

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(item, nil)
	env.listingRepo.On("GetActiveListingByStashItemID", ctx, testStashItemID).Return(listing, nil)

	// Quantity decremented by unlisted amount (2), listings NOT cancelled
	env.stashRepo.On("DecrementQuantity", ctx, testStashItemID, 2).Return(nil)

	err := env.svc.Delete(ctx, testStashItemID, testUserID, boolPtr(false))

	assert.NoError(t, err)
	// Listings should NOT be cancelled
	env.listingRepo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
	env.listingRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	// Item should NOT be hard-deleted
	env.stashRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	env.stashRepo.AssertExpectations(t)
}

func TestStashDelete_UnlistedOnly_NothingListed_DeletesItem(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	// Rune x5, nothing listed
	item := testStashItem(testStashItemID, testUserID, withStashCategory("rune"), withStashQuantity(5))

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(item, nil)
	env.listingRepo.On("GetActiveListingByStashItemID", ctx, testStashItemID).Return(nil, sql.ErrNoRows)

	// Nothing listed → all unlisted → full delete
	env.stashRepo.On("Delete", ctx, testStashItemID).Return(nil)

	err := env.svc.Delete(ctx, testStashItemID, testUserID, boolPtr(false))

	assert.NoError(t, err)
	env.stashRepo.AssertExpectations(t)
}

func TestStashDelete_NonStackable_Listed_DeleteAll(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	// Non-stackable item (helms, qty=1) that is listed
	item := testStashItem(testStashItemID, testUserID, withStashCategory("helms"))
	listing := testListing("listing-1", testUserID, withStashStashItemID(&item.ID), withListingAmount(1))

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(item, nil)
	env.listingRepo.On("GetActiveListingByStashItemID", ctx, testStashItemID).Return(listing, nil)

	env.listingRepo.On("GetByID", ctx, "listing-1").Return(listing, nil)
	env.listingRepo.On("Update", ctx, mock.MatchedBy(func(l *models.Listing) bool {
		return l.ID == "listing-1" && l.Status == "cancelled"
	})).Return(nil)

	// qty=1, listed=1, unlisted=0 → full delete
	env.stashRepo.On("Delete", ctx, testStashItemID).Return(nil)

	err := env.svc.Delete(ctx, testStashItemID, testUserID, boolPtr(true))

	assert.NoError(t, err)
	env.stashRepo.AssertExpectations(t)
	env.listingRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Unlist
// ---------------------------------------------------------------------------

func TestStashUnlist_Success(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID)
	listing := testListing("listing-1", testUserID, withStashStashItemID(&item.ID))

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(item, nil)
	env.listingRepo.On("GetActiveListingByStashItemID", ctx, testStashItemID).Return(listing, nil)
	env.listingRepo.On("GetByID", ctx, "listing-1").Return(listing, nil)
	env.listingRepo.On("Update", ctx, mock.MatchedBy(func(l *models.Listing) bool {
		return l.ID == "listing-1" && l.Status == "cancelled"
	})).Return(nil)

	err := env.svc.Unlist(ctx, testStashItemID, testUserID)

	assert.NoError(t, err)
	env.stashRepo.AssertExpectations(t)
	env.listingRepo.AssertExpectations(t)
	// Stash item should NOT be deleted
	env.stashRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestStashUnlist_NotListed(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID)
	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(item, nil)
	env.listingRepo.On("GetActiveListingByStashItemID", ctx, testStashItemID).Return(nil, sql.ErrNoRows)

	err := env.svc.Unlist(ctx, testStashItemID, testUserID)

	assert.Error(t, err)
	assert.Equal(t, ErrNotListed, err)
}

func TestStashUnlist_NotFound(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(nil, sql.ErrNoRows)

	err := env.svc.Unlist(ctx, testStashItemID, testUserID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestStashUnlist_WrongUser(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testBuyerID).Return(nil, sql.ErrNoRows)

	err := env.svc.Unlist(ctx, testStashItemID, testBuyerID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

// ---------------------------------------------------------------------------
// AdjustQuantity
// ---------------------------------------------------------------------------

func TestStashAdjustQuantity_Increment_Success(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID, withStashCategory("rune"), withStashQuantity(3))
	updated := testStashItem(testStashItemID, testUserID, withStashCategory("rune"), withStashQuantity(5))

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(item, nil).Once()
	env.stashRepo.On("IncrementQuantity", ctx, testStashItemID, 2).Return(nil)
	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(updated, nil).Once()

	result, err := env.svc.AdjustQuantity(ctx, testStashItemID, testUserID, 2)

	assert.NoError(t, err)
	assert.Equal(t, 5, result.Quantity)
	env.stashRepo.AssertExpectations(t)
}

func TestStashAdjustQuantity_Decrement_Success(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID, withStashCategory("gem"), withStashQuantity(5))
	updated := testStashItem(testStashItemID, testUserID, withStashCategory("gem"), withStashQuantity(3))

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(item, nil).Once()
	env.stashRepo.On("DecrementQuantity", ctx, testStashItemID, 2).Return(nil)
	env.stashRepo.On("DeleteIfZeroQuantity", ctx, testStashItemID).Return(nil)
	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(updated, nil).Once()

	result, err := env.svc.AdjustQuantity(ctx, testStashItemID, testUserID, -2)

	assert.NoError(t, err)
	assert.Equal(t, 3, result.Quantity)
	env.stashRepo.AssertExpectations(t)
}

func TestStashAdjustQuantity_DecrementToZero_ItemDeleted(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID, withStashCategory("rune"), withStashQuantity(2))

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(item, nil).Once()
	env.stashRepo.On("DecrementQuantity", ctx, testStashItemID, 2).Return(nil)
	env.stashRepo.On("DeleteIfZeroQuantity", ctx, testStashItemID).Return(nil)
	// Item deleted, re-fetch returns not found
	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(nil, sql.ErrNoRows).Once()

	result, err := env.svc.AdjustQuantity(ctx, testStashItemID, testUserID, -2)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, sql.ErrNoRows)
	env.stashRepo.AssertExpectations(t)
}

func TestStashAdjustQuantity_NonStackable_Rejected(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID, withStashCategory("helms"))

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(item, nil)

	result, err := env.svc.AdjustQuantity(ctx, testStashItemID, testUserID, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrInvalidState, err)
	env.stashRepo.AssertNotCalled(t, "IncrementQuantity", mock.Anything, mock.Anything, mock.Anything)
}

func TestStashAdjustQuantity_NotFound(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(nil, sql.ErrNoRows)

	result, err := env.svc.AdjustQuantity(ctx, testStashItemID, testUserID, 3)

	assert.Error(t, err)
	assert.Nil(t, result)
	env.stashRepo.AssertExpectations(t)
}

func TestStashAdjustQuantity_DecrementError(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID, withStashCategory("misc"), withStashQuantity(1))

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(item, nil)
	env.stashRepo.On("DecrementQuantity", ctx, testStashItemID, 5).Return(errors.New("insufficient quantity"))

	result, err := env.svc.AdjustQuantity(ctx, testStashItemID, testUserID, -5)

	assert.Error(t, err)
	assert.Nil(t, result)
	env.stashRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestStashList_Success(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	items := []*models.StashItem{
		testStashItem(testStashItemID, testUserID, withStashName("Shako")),
		testStashItem(testStashItemID2, testUserID, withStashCategory("rune"), withStashItemType("Ber Rune"), withStashQuantity(3)),
	}

	env.stashRepo.On("ListByUserID", ctx, testUserID, mock.AnythingOfType("repository.StashFilter")).Return(items, 2, nil)

	req := &dto.StashFilterRequest{}
	result, count, err := env.svc.List(ctx, testUserID, req)

	assert.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.Len(t, result, 2)
	env.stashRepo.AssertExpectations(t)
}

func TestStashList_WithFilters(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	env.stashRepo.On("ListByUserID", ctx, testUserID, mock.MatchedBy(func(f repository.StashFilter) bool {
		return f.Query == "shako" &&
			len(f.Categories) == 1 && f.Categories[0] == "helms" &&
			len(f.Rarities) == 1 && f.Rarities[0] == "unique" &&
			f.SortBy == "name" &&
			f.SortOrder == "asc"
	})).Return([]*models.StashItem{}, 0, nil)

	req := &dto.StashFilterRequest{
		Q:          "shako",
		Categories: "helms",
		Rarities:   "unique",
		SortBy:     "name",
		SortOrder:  "asc",
	}

	result, count, err := env.svc.List(ctx, testUserID, req)

	assert.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Len(t, result, 0)
	env.stashRepo.AssertExpectations(t)
}

func TestStashList_WithAffixFilters(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	env.stashRepo.On("ListByUserID", ctx, testUserID, mock.MatchedBy(func(f repository.StashFilter) bool {
		return len(f.AffixFilters) == 1 &&
			f.AffixFilters[0].Code == "all_skills" &&
			f.AffixFilters[0].MinValue != nil && *f.AffixFilters[0].MinValue == 2
	})).Return([]*models.StashItem{}, 0, nil)

	minVal := 2
	filters, _ := json.Marshal([]dto.AffixFilter{{Code: "all_skills", MinValue: &minVal}})

	req := &dto.StashFilterRequest{
		AffixFilters: string(filters),
	}

	_, _, err := env.svc.List(ctx, testUserID, req)

	assert.NoError(t, err)
	env.stashRepo.AssertExpectations(t)
}

func TestStashList_RepoError(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	env.stashRepo.On("ListByUserID", ctx, testUserID, mock.AnythingOfType("repository.StashFilter")).Return(nil, 0, errors.New("db error"))

	req := &dto.StashFilterRequest{}
	result, count, err := env.svc.List(ctx, testUserID, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, count)
	env.stashRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// CreateListingFromStash
// ---------------------------------------------------------------------------

func TestStashCreateListing_Success(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID, withStashName("Shako"))
	profile := testProfile(testUserID, withPremium)

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(item, nil)
	env.stashRepo.On("GetListedAmountForStashItem", ctx, testStashItemID).Return(0, nil)
	env.profileRepo.On("GetByID", mock.Anything, testUserID).Return(profile, nil)
	env.listingRepo.On("Create", ctx, mock.MatchedBy(func(l *models.Listing) bool {
		// Verify the listing has StashItemID set BEFORE repo.Create (no duplicate auto-stash)
		return l.StashItemID != nil && *l.StashItemID == testStashItemID
	})).Return(nil)

	req := &dto.CreateListingFromStashRequest{
		StashItemID: testStashItemID,
		AskingPrice: "2 Ist",
	}

	listing, err := env.svc.CreateListingFromStash(ctx, testUserID, req)

	assert.NoError(t, err)
	assert.NotNil(t, listing)
	assert.Equal(t, "Shako", listing.Name) // displayName fallback to Name
	assert.Equal(t, testStashItemID, *listing.StashItemID)
	// Verify no auto-stash creation happened (stashRepo.Create should NOT be called)
	env.stashRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	env.stashRepo.AssertExpectations(t)
	env.listingRepo.AssertExpectations(t)
}

func TestStashCreateListing_StackableWithAmount(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID, withStashCategory("rune"), withStashItemType("Ber Rune"), withStashQuantity(5))
	profile := testProfile(testUserID, withPremium)

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(item, nil)
	env.stashRepo.On("GetListedAmountForStashItem", ctx, testStashItemID).Return(1, nil) // 1 already listed
	env.profileRepo.On("GetByID", mock.Anything, testUserID).Return(profile, nil)
	env.listingRepo.On("Create", ctx, mock.MatchedBy(func(l *models.Listing) bool {
		return l.StashItemID != nil && *l.StashItemID == testStashItemID
	})).Return(nil)

	amount := 3 // 5 total - 1 listed = 4 available, listing 3 is fine
	req := &dto.CreateListingFromStashRequest{
		StashItemID: testStashItemID,
		Amount:      &amount,
	}

	listing, err := env.svc.CreateListingFromStash(ctx, testUserID, req)

	assert.NoError(t, err)
	assert.NotNil(t, listing)
	env.stashRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	env.stashRepo.AssertExpectations(t)
}

func TestStashCreateListing_InsufficientQuantity(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID, withStashCategory("rune"), withStashQuantity(5))

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(item, nil)
	env.stashRepo.On("GetListedAmountForStashItem", ctx, testStashItemID).Return(3, nil) // 5-3=2 available

	amount := 3 // trying to list 3 but only 2 available
	req := &dto.CreateListingFromStashRequest{
		StashItemID: testStashItemID,
		Amount:      &amount,
	}

	listing, err := env.svc.CreateListingFromStash(ctx, testUserID, req)

	assert.Error(t, err)
	assert.Nil(t, listing)
	assert.Equal(t, ErrInsufficientQuantity, err)
	env.listingRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestStashCreateListing_AllAlreadyListed(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID, withStashCategory("gem"), withStashQuantity(3))

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(item, nil)
	env.stashRepo.On("GetListedAmountForStashItem", ctx, testStashItemID).Return(3, nil) // all listed

	req := &dto.CreateListingFromStashRequest{
		StashItemID: testStashItemID,
	}

	listing, err := env.svc.CreateListingFromStash(ctx, testUserID, req)

	assert.Error(t, err)
	assert.Nil(t, listing)
	assert.Equal(t, ErrInsufficientQuantity, err)
}

func TestStashCreateListing_StashItemNotFound(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(nil, sql.ErrNoRows)

	req := &dto.CreateListingFromStashRequest{
		StashItemID: testStashItemID,
	}

	listing, err := env.svc.CreateListingFromStash(ctx, testUserID, req)

	assert.Error(t, err)
	assert.Nil(t, listing)
	env.stashRepo.AssertExpectations(t)
}

func TestStashCreateListing_WrongOwner(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testBuyerID).Return(nil, sql.ErrNoRows)

	req := &dto.CreateListingFromStashRequest{
		StashItemID: testStashItemID,
	}

	listing, err := env.svc.CreateListingFromStash(ctx, testBuyerID, req)

	assert.Error(t, err)
	assert.Nil(t, listing)
}

func TestStashCreateListing_FreeUserAtLimit(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID, withStashName("Shako"))
	profile := testProfile(testUserID) // free user

	env.stashRepo.On("GetByIDAndUserID", ctx, testStashItemID, testUserID).Return(item, nil)
	env.stashRepo.On("GetListedAmountForStashItem", ctx, testStashItemID).Return(0, nil)
	env.profileRepo.On("GetByID", mock.Anything, testUserID).Return(profile, nil)
	env.listingRepo.On("CountActiveBySellerID", mock.Anything, testUserID).Return(10, nil) // at limit

	req := &dto.CreateListingFromStashRequest{
		StashItemID: testStashItemID,
	}

	listing, err := env.svc.CreateListingFromStash(ctx, testUserID, req)

	assert.Error(t, err)
	assert.Nil(t, listing)
	assert.Equal(t, ErrListingLimitReached, err)
}

// ---------------------------------------------------------------------------
// BulkImportConfirm
// ---------------------------------------------------------------------------

func TestStashBulkImportConfirm_Success(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	env.stashRepo.On("CreateBatch", ctx, mock.AnythingOfType("[]*models.StashItem")).Return(nil)

	items := []dto.CreateStashItemRequest{
		{
			Name:      "Shako",
			ItemType:  "Shako",
			Rarity:    "unique",
			Category:  "helms",
			Game:      "diablo2",
			Platforms: []string{"pc"},
			Region:    "americas",
		},
		{
			ItemType:  "Ber Rune",
			Rarity:    "normal",
			Category:  "rune",
			Game:      "diablo2",
			Platforms: []string{"pc"},
			Region:    "americas",
		},
	}

	result, err := env.svc.BulkImportConfirm(ctx, testUserID, items)

	assert.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "import", result[0].Source)
	assert.Equal(t, "import", result[1].Source)
	assert.Equal(t, "Shako", *result[0].Name)
	assert.Nil(t, result[1].Name)
	env.stashRepo.AssertExpectations(t)
}

func TestStashBulkImportConfirm_EmptyItems(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	result, err := env.svc.BulkImportConfirm(ctx, testUserID, []dto.CreateStashItemRequest{})

	assert.NoError(t, err)
	assert.Len(t, result, 0)
	env.stashRepo.AssertNotCalled(t, "CreateBatch", mock.Anything, mock.Anything)
}

func TestStashBulkImportConfirm_StackableQuantity(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	env.stashRepo.On("CreateBatch", ctx, mock.AnythingOfType("[]*models.StashItem")).Return(nil)

	qty := 10
	items := []dto.CreateStashItemRequest{
		{
			ItemType:  "Amethyst",
			Rarity:    "normal",
			Category:  "gem",
			Game:      "diablo2",
			Quantity:  &qty,
			Platforms: []string{"pc"},
			Region:    "americas",
		},
	}

	result, err := env.svc.BulkImportConfirm(ctx, testUserID, items)

	assert.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, 10, result[0].Quantity)
}

func TestStashBulkImportConfirm_NonStackableIgnoresQuantity(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	env.stashRepo.On("CreateBatch", ctx, mock.AnythingOfType("[]*models.StashItem")).Return(nil)

	qty := 5
	items := []dto.CreateStashItemRequest{
		{
			ItemType:  "Shako",
			Rarity:    "unique",
			Category:  "helms",
			Game:      "diablo2",
			Quantity:  &qty,
			Platforms: []string{"pc"},
			Region:    "americas",
		},
	}

	result, err := env.svc.BulkImportConfirm(ctx, testUserID, items)

	assert.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, 1, result[0].Quantity) // forced to 1
}

func TestStashBulkImportConfirm_RepoError(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	env.stashRepo.On("CreateBatch", ctx, mock.AnythingOfType("[]*models.StashItem")).Return(errors.New("db error"))

	items := []dto.CreateStashItemRequest{
		{
			ItemType:  "Ring",
			Rarity:    "rare",
			Category:  "rings",
			Game:      "diablo2",
			Platforms: []string{"pc"},
			Region:    "americas",
		},
	}

	result, err := env.svc.BulkImportConfirm(ctx, testUserID, items)

	assert.Error(t, err)
	assert.Nil(t, result)
	env.stashRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// BulkImportPreview
// ---------------------------------------------------------------------------

func TestStashBulkImportPreview_VisionNotConfigured(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	// svc has no vision client (empty API key in newStashTestEnv)
	result, err := env.svc.BulkImportPreview(ctx, testUserID, nil, "diablo2", false, false, "americas", []string{"pc"})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrVisionNotConfigured, err)
}

// ---------------------------------------------------------------------------
// HandleTradeCompletion
// ---------------------------------------------------------------------------

func TestStashHandleTradeCompletion_FullFlow(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	stashID := testStashItemID
	listing := testListing(testListingID, testSellerID, withStashStashItemID(&stashID))

	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)

	catalogID := "catalog-ber"
	offeredItems := json.RawMessage(`[{"id":"item1","name":"Ber Rune","type":"rune","quantity":1,"stashItemId":"buyer-stash-1"},{"id":"item2","name":"Jah Rune","type":"rune","quantity":2}]`)
	listingID := testListingID
	offer := testOffer(testOfferID, testBuyerID, &listingID)
	offer.OfferedItems = offeredItems

	// 1. Seller's stash: decrement quantity for listed item
	env.stashRepo.On("DecrementQuantity", ctx, stashID, 1).Return(nil)
	env.stashRepo.On("DeleteIfZeroQuantity", ctx, stashID).Return(nil)

	// 2. Buyer receives: listing is helm (not stackable), no FindMatchingForStack
	env.stashRepo.On("Create", ctx, mock.MatchedBy(func(item *models.StashItem) bool {
		return item.UserID == testBuyerID && item.Source == "trade" && item.ItemType == listing.ItemType
	})).Return(nil).Once()

	// 3. Offered item "Ber Rune" has stashItemId → fetch source stash for fields
	berSource := testStashItem("buyer-stash-1", testBuyerID, withStashCategory("runes"), withStashItemType("rune"), withStashName("Ber Rune"))
	berSource.CatalogItemID = &catalogID
	env.stashRepo.On("GetByID", ctx, "buyer-stash-1").Return(berSource, nil)
	env.stashRepo.On("DecrementQuantity", ctx, "buyer-stash-1", 1).Return(nil)
	env.stashRepo.On("DeleteIfZeroQuantity", ctx, "buyer-stash-1").Return(nil)

	// 4. Seller receives: rune items → FindMatchingForStack returns no match → Create
	// Ber Rune gets catalogItemID from source stash
	env.stashRepo.On("FindMatchingForStack", ctx, testSellerID, "Ber Rune", "rune", "runes", &catalogID).Return(nil, sql.ErrNoRows)
	// Jah Rune has no stashItemId, no source stash → nil catalogItemID
	env.stashRepo.On("FindMatchingForStack", ctx, testSellerID, "Jah Rune", "rune", "runes", (*string)(nil)).Return(nil, sql.ErrNoRows)
	env.stashRepo.On("Create", ctx, mock.MatchedBy(func(item *models.StashItem) bool {
		return item.UserID == testSellerID && item.Source == "trade" && item.Category == "runes"
	})).Return(nil).Twice()

	env.svc.HandleTradeCompletion(ctx, trade, listing, offer)

	env.stashRepo.AssertExpectations(t)
}

func TestStashHandleTradeCompletion_NoStashItemID(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	listing := testListing(testListingID, testSellerID) // no StashItemID, category=helm (not stackable)
	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)
	listingID := testListingID
	offer := testOffer(testOfferID, testBuyerID, &listingID)
	offer.OfferedItems = json.RawMessage(`[]`)

	// Should still create buyer's stash item from listing (helm not stackable)
	env.stashRepo.On("Create", ctx, mock.MatchedBy(func(item *models.StashItem) bool {
		return item.UserID == testBuyerID && item.Source == "trade"
	})).Return(nil)

	env.svc.HandleTradeCompletion(ctx, trade, listing, offer)

	// DecrementQuantity should NOT have been called for seller (no StashItemID)
	env.stashRepo.AssertNotCalled(t, "DecrementQuantity", ctx, mock.Anything, mock.Anything)
	env.stashRepo.AssertExpectations(t)
}

func TestStashHandleTradeCompletion_NilOffer(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	stashID := testStashItemID
	listing := testListing(testListingID, testSellerID, withStashStashItemID(&stashID))
	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)

	// Seller stash decrement
	env.stashRepo.On("DecrementQuantity", ctx, stashID, 1).Return(nil)
	env.stashRepo.On("DeleteIfZeroQuantity", ctx, stashID).Return(nil)

	// Buyer receives (helm not stackable)
	env.stashRepo.On("Create", ctx, mock.MatchedBy(func(item *models.StashItem) bool {
		return item.UserID == testBuyerID && item.Source == "trade"
	})).Return(nil)

	env.svc.HandleTradeCompletion(ctx, trade, listing, nil)

	// No offered items processing should happen
	env.stashRepo.AssertExpectations(t)
}

func TestStashHandleTradeCompletion_OfferedItemWithoutStashLink(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	listing := testListing(testListingID, testSellerID) // helm not stackable
	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)

	// Offered item without stashItemId — no decrement for buyer, but seller still receives
	offeredItems := json.RawMessage(`[{"id":"item1","name":"Ist Rune","type":"rune","quantity":1}]`)
	listingID := testListingID
	offer := testOffer(testOfferID, testBuyerID, &listingID)
	offer.OfferedItems = offeredItems

	// Buyer receives listing item (helm not stackable)
	env.stashRepo.On("Create", ctx, mock.MatchedBy(func(item *models.StashItem) bool {
		return item.UserID == testBuyerID && item.Source == "trade"
	})).Return(nil).Once()

	// Seller receives offered rune (no stashItemId → nil catalogItemID) → FindMatchingForStack returns no match → Create
	env.stashRepo.On("FindMatchingForStack", ctx, testSellerID, "Ist Rune", "rune", "runes", (*string)(nil)).Return(nil, sql.ErrNoRows)
	env.stashRepo.On("Create", ctx, mock.MatchedBy(func(item *models.StashItem) bool {
		return item.UserID == testSellerID && item.Source == "trade" && *item.Name == "Ist Rune" && item.Category == "runes"
	})).Return(nil).Once()

	env.svc.HandleTradeCompletion(ctx, trade, listing, offer)

	// DecrementQuantity should NOT have been called (no stashItemId on offered items, no stashItemID on listing)
	env.stashRepo.AssertNotCalled(t, "DecrementQuantity", mock.Anything, mock.Anything, mock.Anything)
	env.stashRepo.AssertExpectations(t)
}

func TestStashHandleTradeCompletion_BuyerStackableItemStacks(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	// Listing is a rune (stackable)
	listing := testListing(testListingID, testSellerID,
		func(l *models.Listing) { l.Category = "runes"; l.ItemType = "rune"; l.Name = "Ber Rune"; l.Rarity = "normal" },
	)
	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)
	listingID := testListingID
	offer := testOffer(testOfferID, testBuyerID, &listingID)
	offer.OfferedItems = json.RawMessage(`[]`)

	// Seller's stash not linked
	// Buyer receives rune → FindMatchingForStack finds existing → IncrementQuantity
	existingBuyerItem := testStashItem("existing-buyer-rune", testBuyerID, withStashCategory("runes"), withStashItemType("rune"), withStashName("Ber Rune"), withStashQuantity(3))
	env.stashRepo.On("FindMatchingForStack", ctx, testBuyerID, "Ber Rune", "rune", "runes", (*string)(nil)).Return(existingBuyerItem, nil)
	env.stashRepo.On("IncrementQuantity", ctx, "existing-buyer-rune", 1).Return(nil)

	env.svc.HandleTradeCompletion(ctx, trade, listing, offer)

	// Create should NOT have been called for buyer (stacked instead)
	env.stashRepo.AssertNotCalled(t, "Create", mock.Anything, mock.MatchedBy(func(item *models.StashItem) bool {
		return item.UserID == testBuyerID
	}))
	env.stashRepo.AssertExpectations(t)
}

func TestStashHandleTradeCompletion_SellerStackableItemStacks(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	listing := testListing(testListingID, testSellerID) // helm not stackable
	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)

	offeredItems := json.RawMessage(`[{"id":"item1","name":"Ber Rune","type":"rune","category":"runes","quantity":2}]`)
	listingID := testListingID
	offer := testOffer(testOfferID, testBuyerID, &listingID)
	offer.OfferedItems = offeredItems

	// Buyer receives listing item (helm not stackable)
	env.stashRepo.On("Create", ctx, mock.MatchedBy(func(item *models.StashItem) bool {
		return item.UserID == testBuyerID && item.Source == "trade"
	})).Return(nil).Once()

	// Seller receives rune (no stashItemId → nil catalogItemID) → FindMatchingForStack finds existing → IncrementQuantity
	existingSellerItem := testStashItem("existing-seller-rune", testSellerID, withStashCategory("runes"), withStashItemType("rune"), withStashName("Ber Rune"), withStashQuantity(5))
	env.stashRepo.On("FindMatchingForStack", ctx, testSellerID, "Ber Rune", "rune", "runes", (*string)(nil)).Return(existingSellerItem, nil)
	env.stashRepo.On("IncrementQuantity", ctx, "existing-seller-rune", 2).Return(nil)

	env.svc.HandleTradeCompletion(ctx, trade, listing, offer)

	// Create should NOT have been called for seller (stacked instead)
	env.stashRepo.AssertNotCalled(t, "Create", mock.Anything, mock.MatchedBy(func(item *models.StashItem) bool {
		return item.UserID == testSellerID
	}))
	env.stashRepo.AssertExpectations(t)
}

func TestStashHandleTradeCompletion_SellerItemHasCorrectFields(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	listing := testListing(testListingID, testSellerID) // helm not stackable
	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)

	catalogID := "catalog-123"
	offeredItems := json.RawMessage(`[{"id":"item1","name":"Shako","type":"unique","category":"helms","rarity":"unique","catalogItemId":"catalog-123","quantity":1}]`)
	listingID := testListingID
	offer := testOffer(testOfferID, testBuyerID, &listingID)
	offer.OfferedItems = offeredItems

	// Buyer receives listing item
	env.stashRepo.On("Create", ctx, mock.MatchedBy(func(item *models.StashItem) bool {
		return item.UserID == testBuyerID
	})).Return(nil).Once()

	// Seller receives unique helm (not stackable) — verify fields
	env.stashRepo.On("Create", ctx, mock.MatchedBy(func(item *models.StashItem) bool {
		return item.UserID == testSellerID &&
			item.Category == "helms" &&
			item.Rarity == "unique" &&
			item.CatalogItemID != nil && *item.CatalogItemID == catalogID &&
			*item.Name == "Shako"
	})).Return(nil).Once()

	env.svc.HandleTradeCompletion(ctx, trade, listing, offer)

	env.stashRepo.AssertExpectations(t)
}

func TestStashHandleTradeCompletion_SellerStacksWithCatalogIDFromSourceStash(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	listing := testListing(testListingID, testSellerID) // helm not stackable
	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)

	// Offered item has stashItemId but no catalogItemId in JSON
	offeredItems := json.RawMessage(`[{"id":"item1","name":"Ohm","type":"rune","quantity":1,"stashItemId":"buyer-ohm-stash"}]`)
	listingID := testListingID
	offer := testOffer(testOfferID, testBuyerID, &listingID)
	offer.OfferedItems = offeredItems

	// Buyer receives listing item (helm not stackable)
	env.stashRepo.On("Create", ctx, mock.MatchedBy(func(item *models.StashItem) bool {
		return item.UserID == testBuyerID && item.Source == "trade"
	})).Return(nil).Once()

	// Source stash item has catalogItemID
	catalogID := "catalog-ohm"
	sourceStash := testStashItem("buyer-ohm-stash", testBuyerID, withStashCategory("runes"), withStashItemType("rune"), withStashName("Ohm"))
	sourceStash.CatalogItemID = &catalogID
	env.stashRepo.On("GetByID", ctx, "buyer-ohm-stash").Return(sourceStash, nil)
	env.stashRepo.On("DecrementQuantity", ctx, "buyer-ohm-stash", 1).Return(nil)
	env.stashRepo.On("DeleteIfZeroQuantity", ctx, "buyer-ohm-stash").Return(nil)

	// Seller already has an Ohm with same catalogItemID → stack
	existingSellerOhm := testStashItem("seller-ohm-stash", testSellerID, withStashCategory("runes"), withStashItemType("rune"), withStashName("Ohm"), withStashQuantity(2))
	existingSellerOhm.CatalogItemID = &catalogID
	env.stashRepo.On("FindMatchingForStack", ctx, testSellerID, "Ohm", "rune", "runes", &catalogID).Return(existingSellerOhm, nil)
	env.stashRepo.On("IncrementQuantity", ctx, "seller-ohm-stash", 1).Return(nil)

	env.svc.HandleTradeCompletion(ctx, trade, listing, offer)

	// Create should NOT have been called for seller (stacked)
	env.stashRepo.AssertNotCalled(t, "Create", mock.Anything, mock.MatchedBy(func(item *models.StashItem) bool {
		return item.UserID == testSellerID
	}))
	env.stashRepo.AssertExpectations(t)
}

func TestStashHandleTradeCompletion_StackNoMatchFallsBackToCreate(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	// Listing is a rune (stackable), buyer receives it
	listing := testListing(testListingID, testSellerID,
		func(l *models.Listing) { l.Category = "runes"; l.ItemType = "rune"; l.Name = "Ber Rune"; l.Rarity = "normal" },
	)
	trade := testTrade(testTradeID, testOfferID, testListingID, testSellerID, testBuyerID)
	listingID := testListingID
	offer := testOffer(testOfferID, testBuyerID, &listingID)
	offer.OfferedItems = json.RawMessage(`[]`)

	// Buyer receives rune → FindMatchingForStack returns no match → falls back to Create
	env.stashRepo.On("FindMatchingForStack", ctx, testBuyerID, "Ber Rune", "rune", "runes", (*string)(nil)).Return(nil, sql.ErrNoRows)
	env.stashRepo.On("Create", ctx, mock.MatchedBy(func(item *models.StashItem) bool {
		return item.UserID == testBuyerID && item.Source == "trade" && item.Category == "runes"
	})).Return(nil).Once()

	env.svc.HandleTradeCompletion(ctx, trade, listing, offer)

	env.stashRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// ToResponse (single item)
// ---------------------------------------------------------------------------

func TestStashToResponse_Unlisted(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID, withStashName("Shako"))
	env.listingRepo.On("GetActiveListingByStashItemID", ctx, testStashItemID).Return(nil, sql.ErrNoRows)

	resp := env.svc.ToResponse(ctx, item)

	assert.Equal(t, testStashItemID, resp.ID)
	assert.Equal(t, "Shako", resp.DisplayName)
	assert.Equal(t, 1, resp.Quantity)
	assert.False(t, resp.IsListed)
	assert.Equal(t, "manual", resp.Source)
	assert.Empty(t, resp.AskingPrice)
	assert.Nil(t, resp.AskingFor)
}

func TestStashToResponse_Listed(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID, withStashCategory("rune"), withStashQuantity(5))
	listing := testListing("listing-1", testUserID,
		withStashStashItemID(&item.ID),
		withListingAmount(3),
		withAskingPrice("2 Ist"),
		withAskingFor(json.RawMessage(`[{"name":"Ber Rune"}]`)),
	)
	env.listingRepo.On("GetActiveListingByStashItemID", ctx, testStashItemID).Return(listing, nil)

	resp := env.svc.ToResponse(ctx, item)

	assert.Equal(t, 5, resp.Quantity) // ToResponse returns full quantity
	assert.True(t, resp.IsListed)     // listedAmount > 0
	assert.Equal(t, "2 Ist", resp.AskingPrice)
	assert.JSONEq(t, `[{"name":"Ber Rune"}]`, string(resp.AskingFor))
}

func TestStashToResponse_DisplayNameFallback(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID) // no name set
	env.listingRepo.On("GetActiveListingByStashItemID", ctx, testStashItemID).Return(nil, sql.ErrNoRows)

	resp := env.svc.ToResponse(ctx, item)

	assert.Equal(t, "", resp.Name)
	assert.Equal(t, "Shako", resp.DisplayName) // falls back to ItemType
}

func TestStashToResponse_Stackable(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID, withStashCategory("rune"), withStashQuantity(10))
	env.listingRepo.On("GetActiveListingByStashItemID", ctx, testStashItemID).Return(nil, sql.ErrNoRows)

	resp := env.svc.ToResponse(ctx, item)

	assert.True(t, resp.IsStackable)
	assert.Equal(t, 10, resp.Quantity)
}

func TestStashToResponse_NonStackable(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID, withStashCategory("helms"))
	env.listingRepo.On("GetActiveListingByStashItemID", ctx, testStashItemID).Return(nil, sql.ErrNoRows)

	resp := env.svc.ToResponse(ctx, item)

	assert.False(t, resp.IsStackable)
	assert.Equal(t, 1, resp.Quantity)
}

// ---------------------------------------------------------------------------
// ToListResponses (split logic)
// ---------------------------------------------------------------------------

func TestStashToListResponses_AllUnlisted(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID, withStashCategory("rune"), withStashQuantity(5))
	env.listingRepo.On("GetActiveListingByStashItemID", ctx, testStashItemID).Return(nil, sql.ErrNoRows)

	responses := env.svc.ToListResponses(ctx, item)

	require.Len(t, responses, 1)
	assert.Equal(t, 5, responses[0].Quantity)
	assert.False(t, responses[0].IsListed)
	assert.Empty(t, responses[0].AskingPrice)
}

func TestStashToListResponses_AllListed(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID, withStashCategory("gem"), withStashQuantity(3))
	listing := testListing("listing-1", testUserID,
		withStashStashItemID(&item.ID),
		withListingAmount(3),
		withAskingPrice("1 Ber"),
	)
	env.listingRepo.On("GetActiveListingByStashItemID", ctx, testStashItemID).Return(listing, nil)

	responses := env.svc.ToListResponses(ctx, item)

	require.Len(t, responses, 1)
	assert.Equal(t, 3, responses[0].Quantity)
	assert.True(t, responses[0].IsListed)
	assert.Equal(t, "1 Ber", responses[0].AskingPrice)
}

func TestStashToListResponses_PartiallyListed_Split(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID, withStashCategory("rune"), withStashQuantity(5))
	listing := testListing("listing-1", testUserID,
		withStashStashItemID(&item.ID),
		withListingAmount(2),
		withAskingPrice("3 Ist"),
	)
	env.listingRepo.On("GetActiveListingByStashItemID", ctx, testStashItemID).Return(listing, nil)

	responses := env.svc.ToListResponses(ctx, item)

	require.Len(t, responses, 2)

	// First row: unlisted portion — no asking price
	assert.Equal(t, testStashItemID, responses[0].ID)
	assert.Equal(t, 3, responses[0].Quantity)
	assert.False(t, responses[0].IsListed)
	assert.Empty(t, responses[0].AskingPrice)

	// Second row: listed portion — has asking price
	assert.Equal(t, testStashItemID, responses[1].ID)
	assert.Equal(t, 2, responses[1].Quantity)
	assert.True(t, responses[1].IsListed)
	assert.Equal(t, "3 Ist", responses[1].AskingPrice)
}

func TestStashToListResponses_NonStackable_Unlisted(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID, withStashCategory("helms"))
	env.listingRepo.On("GetActiveListingByStashItemID", ctx, testStashItemID).Return(nil, sql.ErrNoRows)

	responses := env.svc.ToListResponses(ctx, item)

	require.Len(t, responses, 1)
	assert.Equal(t, 1, responses[0].Quantity)
	assert.False(t, responses[0].IsListed)
	assert.False(t, responses[0].IsStackable)
}

func TestStashToListResponses_NonStackable_Listed(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID, withStashCategory("helms"))
	listing := testListing("listing-1", testUserID,
		withStashStashItemID(&item.ID),
		withListingAmount(1),
		withAskingPrice("Enigma"),
	)
	env.listingRepo.On("GetActiveListingByStashItemID", ctx, testStashItemID).Return(listing, nil)

	responses := env.svc.ToListResponses(ctx, item)

	require.Len(t, responses, 1)
	assert.Equal(t, 1, responses[0].Quantity)
	assert.True(t, responses[0].IsListed)
	assert.Equal(t, "Enigma", responses[0].AskingPrice)
}

func TestStashToListResponses_SameID_BothRows(t *testing.T) {
	env := newStashTestEnv()
	ctx := context.Background()

	item := testStashItem(testStashItemID, testUserID, withStashCategory("misc"), withStashQuantity(10))
	listing := testListing("listing-1", testUserID,
		withStashStashItemID(&item.ID),
		withListingAmount(4),
		withAskingPrice("5 Ist"),
	)
	env.listingRepo.On("GetActiveListingByStashItemID", ctx, testStashItemID).Return(listing, nil)

	responses := env.svc.ToListResponses(ctx, item)

	require.Len(t, responses, 2)
	// Both rows share the same ID
	assert.Equal(t, responses[0].ID, responses[1].ID)
	// Quantities add up to total
	assert.Equal(t, 10, responses[0].Quantity+responses[1].Quantity)
	// Only listed row has asking price
	assert.Empty(t, responses[0].AskingPrice)
	assert.Equal(t, "5 Ist", responses[1].AskingPrice)
}

// ---------------------------------------------------------------------------
// Model: IsStackable
// ---------------------------------------------------------------------------

func TestStashItem_IsStackable(t *testing.T) {
	tests := []struct {
		category  string
		stackable bool
	}{
		{"rune", true},
		{"gem", true},
		{"misc", true},
		{"helms", false},
		{"weapons", false},
		{"rings", false},
		{"amulets", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.category, func(t *testing.T) {
			item := &models.StashItem{Category: tc.category}
			assert.Equal(t, tc.stackable, item.IsStackable())
		})
	}
}

// ---------------------------------------------------------------------------
// Model: GetDisplayName
// ---------------------------------------------------------------------------

func TestStashItem_GetDisplayName(t *testing.T) {
	t.Run("with name", func(t *testing.T) {
		name := "Harlequin Crest"
		item := &models.StashItem{Name: &name, ItemType: "Shako"}
		assert.Equal(t, "Harlequin Crest", item.GetDisplayName())
	})

	t.Run("without name", func(t *testing.T) {
		item := &models.StashItem{ItemType: "Shako"}
		assert.Equal(t, "Shako", item.GetDisplayName())
	})

	t.Run("empty name", func(t *testing.T) {
		name := ""
		item := &models.StashItem{Name: &name, ItemType: "Ring"}
		assert.Equal(t, "Ring", item.GetDisplayName())
	})
}

// ---------------------------------------------------------------------------
// Model: nullable getters
// ---------------------------------------------------------------------------

func TestStashItem_NullableGetters(t *testing.T) {
	// All nil
	item := &models.StashItem{}
	assert.Equal(t, "", item.GetName())
	assert.Equal(t, "", item.GetImageURL())
	assert.Equal(t, "", item.GetRuneOrder())
	assert.Equal(t, "", item.GetBaseItemCode())
	assert.Equal(t, "", item.GetBaseItemName())
	assert.Equal(t, "", item.GetCatalogItemID())

	// All set
	name := "Test"
	url := "https://img.png"
	ro := "r01r02"
	bic := "abc"
	bin := "Base"
	cid := "cat-1"
	item2 := &models.StashItem{
		Name:          &name,
		ImageURL:      &url,
		RuneOrder:     &ro,
		BaseItemCode:  &bic,
		BaseItemName:  &bin,
		CatalogItemID: &cid,
	}
	assert.Equal(t, "Test", item2.GetName())
	assert.Equal(t, "https://img.png", item2.GetImageURL())
	assert.Equal(t, "r01r02", item2.GetRuneOrder())
	assert.Equal(t, "abc", item2.GetBaseItemCode())
	assert.Equal(t, "Base", item2.GetBaseItemName())
	assert.Equal(t, "cat-1", item2.GetCatalogItemID())
}

// ---------------------------------------------------------------------------
// parseCSV helper
// ---------------------------------------------------------------------------

func TestParseCSV(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"helms", []string{"helms"}},
		{"helms,weapons,runes", []string{"helms", "weapons", "runes"}},
		{"helms, weapons , runes", []string{"helms", "weapons", "runes"}},
		{",helms,,weapons,", []string{"helms", "weapons"}},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := parseCSV(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// isImageContentType helper
// ---------------------------------------------------------------------------

func TestIsImageContentType(t *testing.T) {
	assert.True(t, isImageContentType("image/jpeg"))
	assert.True(t, isImageContentType("image/png"))
	assert.True(t, isImageContentType("image/gif"))
	assert.True(t, isImageContentType("image/webp"))
	assert.False(t, isImageContentType("application/json"))
	assert.False(t, isImageContentType("text/plain"))
	assert.False(t, isImageContentType(""))
}
