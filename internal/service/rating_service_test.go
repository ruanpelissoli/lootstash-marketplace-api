package service

import (
	"context"
	"errors"
	"testing"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/models"
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/repository/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// helper to build the full RatingService with mock dependencies.
func newTestRatingService() (
	*RatingService,
	*mocks.MockRatingRepository,
	*mocks.MockTransactionRepository,
	*mocks.MockNotificationRepository,
	*mocks.MockProfileRepository,
) {
	ratingRepo := new(mocks.MockRatingRepository)
	txnRepo := new(mocks.MockTransactionRepository)
	notifRepo := new(mocks.MockNotificationRepository)
	profileRepo := new(mocks.MockProfileRepository)

	notifSvc := NewNotificationService(notifRepo, newTestRedis())
	profileSvc := NewProfileService(profileRepo, newTestRedis(), nil)
	svc := NewRatingService(ratingRepo, txnRepo, profileSvc, notifSvc)

	return svc, ratingRepo, txnRepo, notifRepo, profileRepo
}

// ---------------------------------------------------------------------------
// Create – happy paths
// ---------------------------------------------------------------------------

func TestRatingCreate_SellerRatesBuyer(t *testing.T) {
	svc, ratingRepo, txnRepo, notifRepo, _ := newTestRatingService()

	txn := testTransaction(testTransactionID, testSellerID, testBuyerID)
	txnRepo.On("GetByID", mock.Anything, testTransactionID).Return(txn, nil)
	ratingRepo.On("Exists", mock.Anything, testTransactionID, testSellerID).Return(false, nil)
	ratingRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Rating")).Return(nil)
	notifRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Notification")).Return(nil)

	req := &dto.CreateRatingRequest{
		TransactionID: testTransactionID,
		Stars:         5,
	}
	rating, err := svc.Create(context.Background(), testSellerID, req)

	assert.NoError(t, err)
	assert.NotNil(t, rating)
	assert.Equal(t, testBuyerID, rating.RatedID)
	assert.Equal(t, testSellerID, rating.RaterID)
	assert.Equal(t, 5, rating.Stars)
	assert.Equal(t, testTransactionID, rating.TransactionID)
	assert.NotEmpty(t, rating.ID)
	assert.Nil(t, rating.Comment)

	ratingRepo.AssertExpectations(t)
	txnRepo.AssertExpectations(t)
	notifRepo.AssertExpectations(t)
}

func TestRatingCreate_BuyerRatesSeller(t *testing.T) {
	svc, ratingRepo, txnRepo, notifRepo, _ := newTestRatingService()

	txn := testTransaction(testTransactionID, testSellerID, testBuyerID)
	txnRepo.On("GetByID", mock.Anything, testTransactionID).Return(txn, nil)
	ratingRepo.On("Exists", mock.Anything, testTransactionID, testBuyerID).Return(false, nil)
	ratingRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Rating")).Return(nil)
	notifRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Notification")).Return(nil)

	req := &dto.CreateRatingRequest{
		TransactionID: testTransactionID,
		Stars:         3,
	}
	rating, err := svc.Create(context.Background(), testBuyerID, req)

	assert.NoError(t, err)
	assert.NotNil(t, rating)
	assert.Equal(t, testSellerID, rating.RatedID)
	assert.Equal(t, testBuyerID, rating.RaterID)
	assert.Equal(t, 3, rating.Stars)

	ratingRepo.AssertExpectations(t)
	txnRepo.AssertExpectations(t)
	notifRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Create – error paths
// ---------------------------------------------------------------------------

func TestRatingCreate_NonParticipant(t *testing.T) {
	svc, _, txnRepo, _, _ := newTestRatingService()

	txn := testTransaction(testTransactionID, testSellerID, testBuyerID)
	txnRepo.On("GetByID", mock.Anything, testTransactionID).Return(txn, nil)

	req := &dto.CreateRatingRequest{
		TransactionID: testTransactionID,
		Stars:         4,
	}
	rating, err := svc.Create(context.Background(), testUserID, req)

	assert.Nil(t, rating)
	assert.ErrorIs(t, err, ErrForbidden)
	txnRepo.AssertExpectations(t)
}

func TestRatingCreate_AlreadyRated(t *testing.T) {
	svc, ratingRepo, txnRepo, _, _ := newTestRatingService()

	txn := testTransaction(testTransactionID, testSellerID, testBuyerID)
	txnRepo.On("GetByID", mock.Anything, testTransactionID).Return(txn, nil)
	ratingRepo.On("Exists", mock.Anything, testTransactionID, testSellerID).Return(true, nil)

	req := &dto.CreateRatingRequest{
		TransactionID: testTransactionID,
		Stars:         5,
	}
	rating, err := svc.Create(context.Background(), testSellerID, req)

	assert.Nil(t, rating)
	assert.ErrorIs(t, err, ErrAlreadyExists)
	ratingRepo.AssertExpectations(t)
	txnRepo.AssertExpectations(t)
}

func TestRatingCreate_TransactionNotFound(t *testing.T) {
	svc, _, txnRepo, _, _ := newTestRatingService()

	txnRepo.On("GetByID", mock.Anything, testTransactionID).Return(nil, ErrNotFound)

	req := &dto.CreateRatingRequest{
		TransactionID: testTransactionID,
		Stars:         4,
	}
	rating, err := svc.Create(context.Background(), testSellerID, req)

	assert.Nil(t, rating)
	assert.ErrorIs(t, err, ErrNotFound)
	txnRepo.AssertExpectations(t)
}

func TestRatingCreate_RepoCreateError(t *testing.T) {
	svc, ratingRepo, txnRepo, _, _ := newTestRatingService()

	txn := testTransaction(testTransactionID, testSellerID, testBuyerID)
	txnRepo.On("GetByID", mock.Anything, testTransactionID).Return(txn, nil)
	ratingRepo.On("Exists", mock.Anything, testTransactionID, testSellerID).Return(false, nil)

	repoErr := errors.New("db connection lost")
	ratingRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Rating")).Return(repoErr)

	req := &dto.CreateRatingRequest{
		TransactionID: testTransactionID,
		Stars:         5,
	}
	rating, err := svc.Create(context.Background(), testSellerID, req)

	assert.Nil(t, rating)
	assert.ErrorIs(t, err, repoErr)
	ratingRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Create – comment handling
// ---------------------------------------------------------------------------

func TestRatingCreate_WithComment(t *testing.T) {
	svc, ratingRepo, txnRepo, notifRepo, _ := newTestRatingService()

	txn := testTransaction(testTransactionID, testSellerID, testBuyerID)
	txnRepo.On("GetByID", mock.Anything, testTransactionID).Return(txn, nil)
	ratingRepo.On("Exists", mock.Anything, testTransactionID, testSellerID).Return(false, nil)
	ratingRepo.On("Create", mock.Anything, mock.MatchedBy(func(r *models.Rating) bool {
		return r.Comment != nil && *r.Comment == "Great trader!"
	})).Return(nil)
	notifRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Notification")).Return(nil)

	req := &dto.CreateRatingRequest{
		TransactionID: testTransactionID,
		Stars:         5,
		Comment:       "Great trader!",
	}
	rating, err := svc.Create(context.Background(), testSellerID, req)

	assert.NoError(t, err)
	assert.NotNil(t, rating)
	assert.NotNil(t, rating.Comment)
	assert.Equal(t, "Great trader!", *rating.Comment)

	ratingRepo.AssertExpectations(t)
}

func TestRatingCreate_WithoutComment(t *testing.T) {
	svc, ratingRepo, txnRepo, notifRepo, _ := newTestRatingService()

	txn := testTransaction(testTransactionID, testSellerID, testBuyerID)
	txnRepo.On("GetByID", mock.Anything, testTransactionID).Return(txn, nil)
	ratingRepo.On("Exists", mock.Anything, testTransactionID, testBuyerID).Return(false, nil)
	ratingRepo.On("Create", mock.Anything, mock.MatchedBy(func(r *models.Rating) bool {
		return r.Comment == nil
	})).Return(nil)
	notifRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Notification")).Return(nil)

	req := &dto.CreateRatingRequest{
		TransactionID: testTransactionID,
		Stars:         4,
		Comment:       "",
	}
	rating, err := svc.Create(context.Background(), testBuyerID, req)

	assert.NoError(t, err)
	assert.NotNil(t, rating)
	assert.Nil(t, rating.Comment)

	ratingRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Create – notification verification
// ---------------------------------------------------------------------------

func TestRatingCreate_NotifiesRatedUser(t *testing.T) {
	svc, ratingRepo, txnRepo, notifRepo, _ := newTestRatingService()

	txn := testTransaction(testTransactionID, testSellerID, testBuyerID)
	txnRepo.On("GetByID", mock.Anything, testTransactionID).Return(txn, nil)
	ratingRepo.On("Exists", mock.Anything, testTransactionID, testSellerID).Return(false, nil)
	ratingRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Rating")).Return(nil)

	// Expect notification to be created for the buyer (the rated user) with correct fields
	notifRepo.On("Create", mock.Anything, mock.MatchedBy(func(n *models.Notification) bool {
		return n.UserID == testBuyerID &&
			n.Type == models.NotificationTypeRatingReceived &&
			n.Title == "New Rating" &&
			n.ReferenceType != nil && *n.ReferenceType == "transaction" &&
			n.ReferenceID != nil && *n.ReferenceID == testTransactionID
	})).Return(nil)

	req := &dto.CreateRatingRequest{
		TransactionID: testTransactionID,
		Stars:         5,
	}
	rating, err := svc.Create(context.Background(), testSellerID, req)

	assert.NoError(t, err)
	assert.NotNil(t, rating)
	notifRepo.AssertExpectations(t)
}

func TestRatingCreate_NotificationErrorDoesNotFail(t *testing.T) {
	svc, ratingRepo, txnRepo, notifRepo, _ := newTestRatingService()

	txn := testTransaction(testTransactionID, testSellerID, testBuyerID)
	txnRepo.On("GetByID", mock.Anything, testTransactionID).Return(txn, nil)
	ratingRepo.On("Exists", mock.Anything, testTransactionID, testSellerID).Return(false, nil)
	ratingRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Rating")).Return(nil)
	// Notification repo returns an error, but Create should still succeed
	notifRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Notification")).Return(errors.New("notification failed"))

	req := &dto.CreateRatingRequest{
		TransactionID: testTransactionID,
		Stars:         4,
	}
	rating, err := svc.Create(context.Background(), testSellerID, req)

	assert.NoError(t, err)
	assert.NotNil(t, rating)
	notifRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// GetByUserID
// ---------------------------------------------------------------------------

func TestRatingGetByUserID(t *testing.T) {
	svc, ratingRepo, _, _, _ := newTestRatingService()

	expected := []*models.Rating{
		testRating(testRatingID, testTransactionID, testSellerID, testBuyerID, 5),
		testRating("rating-2", "txn-2", testBuyerID, testSellerID, 4),
	}
	ratingRepo.On("GetByUserID", mock.Anything, testBuyerID, 0, 20).Return(expected, 2, nil)

	ratings, total, err := svc.GetByUserID(context.Background(), testBuyerID, 0, 20)

	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, ratings, 2)
	assert.Equal(t, expected, ratings)
	ratingRepo.AssertExpectations(t)
}

func TestRatingGetByUserID_Empty(t *testing.T) {
	svc, ratingRepo, _, _, _ := newTestRatingService()

	ratingRepo.On("GetByUserID", mock.Anything, testUserID, 0, 10).Return([]*models.Rating{}, 0, nil)

	ratings, total, err := svc.GetByUserID(context.Background(), testUserID, 0, 10)

	assert.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, ratings)
	ratingRepo.AssertExpectations(t)
}

func TestRatingGetByUserID_RepoError(t *testing.T) {
	svc, ratingRepo, _, _, _ := newTestRatingService()

	ratingRepo.On("GetByUserID", mock.Anything, testUserID, 0, 10).Return(nil, 0, errors.New("db error"))

	ratings, total, err := svc.GetByUserID(context.Background(), testUserID, 0, 10)

	assert.Error(t, err)
	assert.Equal(t, 0, total)
	assert.Nil(t, ratings)
	ratingRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// GetByTransactionID
// ---------------------------------------------------------------------------

func TestRatingGetByTransactionID(t *testing.T) {
	svc, ratingRepo, _, _, _ := newTestRatingService()

	expected := []*models.Rating{
		testRating(testRatingID, testTransactionID, testSellerID, testBuyerID, 5),
		testRating("rating-2", testTransactionID, testBuyerID, testSellerID, 4),
	}
	ratingRepo.On("GetByTransactionID", mock.Anything, testTransactionID).Return(expected, nil)

	ratings, err := svc.GetByTransactionID(context.Background(), testTransactionID)

	assert.NoError(t, err)
	assert.Len(t, ratings, 2)
	assert.Equal(t, expected, ratings)
	ratingRepo.AssertExpectations(t)
}

func TestRatingGetByTransactionID_Empty(t *testing.T) {
	svc, ratingRepo, _, _, _ := newTestRatingService()

	ratingRepo.On("GetByTransactionID", mock.Anything, testTransactionID).Return([]*models.Rating{}, nil)

	ratings, err := svc.GetByTransactionID(context.Background(), testTransactionID)

	assert.NoError(t, err)
	assert.Empty(t, ratings)
	ratingRepo.AssertExpectations(t)
}

func TestRatingGetByTransactionID_RepoError(t *testing.T) {
	svc, ratingRepo, _, _, _ := newTestRatingService()

	ratingRepo.On("GetByTransactionID", mock.Anything, testTransactionID).Return(nil, errors.New("db error"))

	ratings, err := svc.GetByTransactionID(context.Background(), testTransactionID)

	assert.Error(t, err)
	assert.Nil(t, ratings)
	ratingRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// ToResponse
// ---------------------------------------------------------------------------

func TestRatingToResponse_WithRater(t *testing.T) {
	svc, _, _, _, _ := newTestRatingService()

	rater := testProfile(testSellerID)
	comment := "Smooth trade"
	rating := &models.Rating{
		ID:            testRatingID,
		TransactionID: testTransactionID,
		RaterID:       testSellerID,
		RatedID:       testBuyerID,
		Stars:         5,
		Comment:       &comment,
		Rater:         rater,
	}

	resp := svc.ToResponse(rating)

	assert.Equal(t, testRatingID, resp.ID)
	assert.Equal(t, testTransactionID, resp.TransactionID)
	assert.Equal(t, testSellerID, resp.RaterID)
	assert.Equal(t, testBuyerID, resp.RatedID)
	assert.Equal(t, 5, resp.Stars)
	assert.Equal(t, "Smooth trade", resp.Comment)
	assert.NotNil(t, resp.Rater)
	assert.Equal(t, testSellerID, resp.Rater.ID)
	assert.Equal(t, rater.Username, resp.Rater.Username)
}

func TestRatingToResponse_WithoutRater(t *testing.T) {
	svc, _, _, _, _ := newTestRatingService()

	rating := &models.Rating{
		ID:            testRatingID,
		TransactionID: testTransactionID,
		RaterID:       testSellerID,
		RatedID:       testBuyerID,
		Stars:         3,
		Comment:       nil,
		Rater:         nil,
	}

	resp := svc.ToResponse(rating)

	assert.Equal(t, testRatingID, resp.ID)
	assert.Equal(t, testTransactionID, resp.TransactionID)
	assert.Equal(t, testSellerID, resp.RaterID)
	assert.Equal(t, testBuyerID, resp.RatedID)
	assert.Equal(t, 3, resp.Stars)
	assert.Equal(t, "", resp.Comment)
	assert.Nil(t, resp.Rater)
}

func TestRatingToResponse_CommentEmptyWhenNil(t *testing.T) {
	svc, _, _, _, _ := newTestRatingService()

	rating := testRating(testRatingID, testTransactionID, testSellerID, testBuyerID, 4)
	// testRating does not set Comment, so it is nil

	resp := svc.ToResponse(rating)

	assert.Equal(t, "", resp.Comment)
}
