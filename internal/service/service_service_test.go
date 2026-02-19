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

func setupServiceService(
	profileRepo *mocks.MockProfileRepository,
	serviceRepo *mocks.MockServiceRepository,
	redis *cache.RedisClient,
) (*ServiceService, *ProfileService) {
	profileService := NewProfileService(profileRepo, redis, nil)
	svc := NewServiceService(serviceRepo, profileService, redis)
	return svc, profileService
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestServiceCreate_Success(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svc, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	ctx := context.Background()
	askingFor := json.RawMessage(`[{"name":"Ist","quantity":1}]`)

	serviceRepo.On("ExistsByProviderAndType", mock.Anything, testProviderID, "rush", "diablo2").Return(false, nil)
	serviceRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Service")).Return(nil)
	profileRepo.On("GetByID", mock.Anything, testProviderID).Return(testProfile(testProviderID), nil)

	req := &dto.CreateServiceRequest{
		ServiceType: "rush",
		Name:        "Normal Rush",
		Description: "Fast normal difficulty rush",
		AskingPrice: "Ist rune",
		AskingFor:   askingFor,
		Notes:       "Available evenings EST",
		Game:        "diablo2",
		Ladder:      true,
		Hardcore:    false,
		IsNonRotw:   false,
		Platforms:   []string{"pc", "xbox"},
		Region:      "americas",
	}

	result, err := svc.Create(ctx, testProviderID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.ID, "UUID should be generated")
	assert.Equal(t, testProviderID, result.ProviderID)
	assert.Equal(t, "rush", result.ServiceType)
	assert.Equal(t, "Normal Rush", result.Name)
	assert.Equal(t, "active", result.Status)
	assert.Equal(t, "diablo2", result.Game)
	assert.True(t, result.Ladder)
	assert.False(t, result.Hardcore)
	assert.False(t, result.IsNonRotw)
	assert.Equal(t, []string{"pc", "xbox"}, result.Platforms)
	assert.Equal(t, "americas", result.Region)
	assert.NotNil(t, result.Description)
	assert.Equal(t, "Fast normal difficulty rush", *result.Description)
	assert.NotNil(t, result.AskingPrice)
	assert.Equal(t, "Ist rune", *result.AskingPrice)
	assert.NotNil(t, result.Notes)
	assert.Equal(t, "Available evenings EST", *result.Notes)
	assert.JSONEq(t, `[{"name":"Ist","quantity":1}]`, string(result.AskingFor))
	assert.False(t, result.CreatedAt.IsZero())
	assert.False(t, result.UpdatedAt.IsZero())
	// Provider should be attached after creation
	assert.NotNil(t, result.Provider)

	serviceRepo.AssertExpectations(t)
	profileRepo.AssertExpectations(t)
}

func TestServiceCreate_DuplicateTypeProviderGame(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svc, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	ctx := context.Background()

	// ExistsByProviderAndType returns true => duplicate
	serviceRepo.On("ExistsByProviderAndType", mock.Anything, testProviderID, "rush", "diablo2").Return(true, nil)

	req := &dto.CreateServiceRequest{
		ServiceType: "rush",
		Name:        "Normal Rush",
		Game:        "diablo2",
		Platforms:   []string{"pc"},
		Region:      "americas",
	}

	result, err := svc.Create(ctx, testProviderID, req)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrAlreadyExists)

	// Create should NOT have been called
	serviceRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	serviceRepo.AssertExpectations(t)
}

func TestServiceCreate_DeduplicatesPlatforms(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svc, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	ctx := context.Background()

	serviceRepo.On("ExistsByProviderAndType", mock.Anything, testProviderID, "rush", "diablo2").Return(false, nil)
	serviceRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Service")).Return(nil)
	profileRepo.On("GetByID", mock.Anything, testProviderID).Return(testProfile(testProviderID), nil)

	req := &dto.CreateServiceRequest{
		ServiceType: "rush",
		Name:        "Normal Rush",
		Game:        "diablo2",
		Platforms:   []string{"pc", "xbox", "pc", "xbox", "pc"},
		Region:      "americas",
	}

	result, err := svc.Create(ctx, testProviderID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, []string{"pc", "xbox"}, result.Platforms, "duplicate platforms should be removed")

	serviceRepo.AssertExpectations(t)
}

func TestServiceCreate_OptionalFieldsEmpty(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svc, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	ctx := context.Background()

	serviceRepo.On("ExistsByProviderAndType", mock.Anything, testProviderID, "rush", "diablo2").Return(false, nil)
	serviceRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Service")).Return(nil)
	profileRepo.On("GetByID", mock.Anything, testProviderID).Return(testProfile(testProviderID), nil)

	req := &dto.CreateServiceRequest{
		ServiceType: "rush",
		Name:        "Normal Rush",
		Description: "", // empty
		AskingPrice: "", // empty
		Notes:       "", // empty
		Game:        "diablo2",
		Platforms:   []string{"pc"},
		Region:      "americas",
	}

	result, err := svc.Create(ctx, testProviderID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Nil(t, result.Description, "empty description should result in nil pointer")
	assert.Nil(t, result.AskingPrice, "empty askingPrice should result in nil pointer")
	assert.Nil(t, result.Notes, "empty notes should result in nil pointer")

	serviceRepo.AssertExpectations(t)
}

func TestServiceCreate_OptionalFieldsSet(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svc, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	ctx := context.Background()

	serviceRepo.On("ExistsByProviderAndType", mock.Anything, testProviderID, "crush", "diablo2").Return(false, nil)
	serviceRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Service")).Return(nil)
	profileRepo.On("GetByID", mock.Anything, testProviderID).Return(testProfile(testProviderID), nil)

	req := &dto.CreateServiceRequest{
		ServiceType: "crush",
		Name:        "Cow Level Crush",
		Description: "Full cow level clear",
		AskingPrice: "Lem rune",
		Notes:       "Only softcore ladder",
		Game:        "diablo2",
		Platforms:   []string{"pc"},
		Region:      "europe",
	}

	result, err := svc.Create(ctx, testProviderID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Description)
	assert.Equal(t, "Full cow level clear", *result.Description)
	assert.NotNil(t, result.AskingPrice)
	assert.Equal(t, "Lem rune", *result.AskingPrice)
	assert.NotNil(t, result.Notes)
	assert.Equal(t, "Only softcore ladder", *result.Notes)

	serviceRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestServiceUpdate_Success(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svc, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	ctx := context.Background()

	existing := testServiceModel(testServiceID, testProviderID)
	serviceRepo.On("GetByID", mock.Anything, testServiceID).Return(existing, nil)
	serviceRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Service")).Return(nil)

	newName := "Hell Rush"
	req := &dto.UpdateServiceRequest{
		Name: &newName,
	}

	result, err := svc.Update(ctx, testServiceID, testProviderID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Hell Rush", result.Name)
	// Other fields should remain unchanged
	assert.Equal(t, "rush", result.ServiceType)
	assert.Equal(t, "diablo2", result.Game)
	assert.Equal(t, []string{"pc"}, result.Platforms)
	assert.Equal(t, "americas", result.Region)

	serviceRepo.AssertExpectations(t)
}

func TestServiceUpdate_NotOwner(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svc, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	ctx := context.Background()
	otherUser := "other-user-999"

	existing := testServiceModel(testServiceID, testProviderID)
	serviceRepo.On("GetByID", mock.Anything, testServiceID).Return(existing, nil)

	newName := "Hijacked"
	req := &dto.UpdateServiceRequest{
		Name: &newName,
	}

	result, err := svc.Update(ctx, testServiceID, otherUser, req)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrForbidden)

	serviceRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	serviceRepo.AssertExpectations(t)
}

func TestServiceUpdate_MultipleFields(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svc, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	ctx := context.Background()

	existing := testServiceModel(testServiceID, testProviderID)
	serviceRepo.On("GetByID", mock.Anything, testServiceID).Return(existing, nil)
	serviceRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Service")).Return(nil)

	newName := "Nightmare Rush"
	newDesc := "Nightmare difficulty rush service"
	newPrice := "Um rune"
	newNotes := "Weekends only"
	newRegion := "europe"
	newAskingFor := json.RawMessage(`[{"name":"Um","quantity":1}]`)

	req := &dto.UpdateServiceRequest{
		Name:        &newName,
		Description: &newDesc,
		AskingPrice: &newPrice,
		AskingFor:   newAskingFor,
		Notes:       &newNotes,
		Platforms:   []string{"pc", "xbox"},
		Region:      &newRegion,
	}

	result, err := svc.Update(ctx, testServiceID, testProviderID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Nightmare Rush", result.Name)
	assert.NotNil(t, result.Description)
	assert.Equal(t, "Nightmare difficulty rush service", *result.Description)
	assert.NotNil(t, result.AskingPrice)
	assert.Equal(t, "Um rune", *result.AskingPrice)
	assert.NotNil(t, result.Notes)
	assert.Equal(t, "Weekends only", *result.Notes)
	assert.Equal(t, []string{"pc", "xbox"}, result.Platforms)
	assert.Equal(t, "europe", result.Region)
	assert.JSONEq(t, `[{"name":"Um","quantity":1}]`, string(result.AskingFor))

	serviceRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestServiceDelete_Success(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svc, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	ctx := context.Background()

	existing := testServiceModel(testServiceID, testProviderID)
	assert.Equal(t, "active", existing.Status, "precondition: service starts as active")

	serviceRepo.On("GetByID", mock.Anything, testServiceID).Return(existing, nil)
	serviceRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Service")).Return(nil)

	err := svc.Delete(ctx, testServiceID, testProviderID)

	assert.NoError(t, err)
	assert.Equal(t, "cancelled", existing.Status, "status should be set to cancelled")

	serviceRepo.AssertExpectations(t)
}

func TestServiceDelete_NotOwner(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svc, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	ctx := context.Background()
	otherUser := "other-user-999"

	existing := testServiceModel(testServiceID, testProviderID)
	serviceRepo.On("GetByID", mock.Anything, testServiceID).Return(existing, nil)

	err := svc.Delete(ctx, testServiceID, otherUser)

	assert.ErrorIs(t, err, ErrForbidden)
	assert.Equal(t, "active", existing.Status, "status should remain unchanged")

	serviceRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	serviceRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestServiceGetByID_Success(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svc, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	ctx := context.Background()
	expected := testServiceModel(testServiceID, testProviderID)

	serviceRepo.On("GetByIDWithProvider", mock.Anything, testServiceID).Return(expected, nil)

	result, err := svc.GetByID(ctx, testServiceID)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)

	serviceRepo.AssertExpectations(t)
}

func TestServiceGetByID_NotFound(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svc, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	ctx := context.Background()

	serviceRepo.On("GetByIDWithProvider", mock.Anything, "nonexistent").Return(nil, ErrNotFound)

	result, err := svc.GetByID(ctx, "nonexistent")

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrNotFound)

	serviceRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// ToServiceResponse
// ---------------------------------------------------------------------------

func TestServiceToServiceResponse_MapsAllFields(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svc, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	desc := "Full rush service"
	price := "Ist rune"
	notes := "Evening hours only"

	service := testServiceModel(testServiceID, testProviderID)
	service.Description = &desc
	service.AskingPrice = &price
	service.Notes = &notes
	service.AskingFor = json.RawMessage(`[{"name":"Ist","quantity":1}]`)
	service.Ladder = true
	service.Hardcore = true
	service.IsNonRotw = true
	service.Platforms = []string{"pc", "xbox"}
	service.Region = "europe"

	resp := svc.ToServiceResponse(service)

	assert.NotNil(t, resp)
	assert.Equal(t, testServiceID, resp.ID)
	assert.Equal(t, "rush", resp.ServiceType)
	assert.Equal(t, "Normal Rush", resp.Name)
	assert.Equal(t, "Full rush service", resp.Description)
	assert.Equal(t, "Ist rune", resp.AskingPrice)
	assert.JSONEq(t, `[{"name":"Ist","quantity":1}]`, string(resp.AskingFor))
	assert.Equal(t, "diablo2", resp.Game)
	assert.True(t, resp.Ladder)
	assert.True(t, resp.Hardcore)
	assert.True(t, resp.IsNonRotw)
	assert.Equal(t, []string{"pc", "xbox"}, resp.Platforms)
	assert.Equal(t, "europe", resp.Region)
	assert.Equal(t, "Evening hours only", resp.Notes)
	assert.Equal(t, "active", resp.Status)
	assert.Equal(t, service.CreatedAt, resp.CreatedAt)
	assert.Equal(t, service.UpdatedAt, resp.UpdatedAt)
}

func TestServiceToServiceResponse_NilOptionalFields(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svc, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	service := testServiceModel(testServiceID, testProviderID)
	// Default testServiceModel has nil Description, AskingPrice, Notes

	resp := svc.ToServiceResponse(service)

	assert.NotNil(t, resp)
	assert.Equal(t, "", resp.Description, "nil Description should become empty string via GetDescription()")
	assert.Equal(t, "", resp.AskingPrice, "nil AskingPrice should become empty string via GetAskingPrice()")
	assert.Equal(t, "", resp.Notes, "nil Notes should become empty string via GetNotes()")
}

// ---------------------------------------------------------------------------
// ToProviderCardResponse
// ---------------------------------------------------------------------------

func TestServiceToProviderCardResponse(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svcService, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	provider := testProfile(testProviderID)

	svc1 := testServiceModel("svc-1", testProviderID)
	svc2 := testServiceModel("svc-2", testProviderID)
	svc2.ServiceType = "crush"
	svc2.Name = "Cow Level Crush"

	desc := "A great service"
	svc2.Description = &desc

	serviceSlice := []*models.Service{svc1, svc2}
	card := svcService.ToProviderCardResponse(provider, serviceSlice)

	// Provider should be mapped
	assert.NotNil(t, card.Provider)
	assert.Equal(t, testProviderID, card.Provider.ID)
	assert.Equal(t, provider.Username, card.Provider.Username)

	// Services should be mapped
	assert.Len(t, card.Services, 2)

	assert.Equal(t, "svc-1", card.Services[0].ID)
	assert.Equal(t, "rush", card.Services[0].ServiceType)
	assert.Equal(t, "Normal Rush", card.Services[0].Name)

	assert.Equal(t, "svc-2", card.Services[1].ID)
	assert.Equal(t, "crush", card.Services[1].ServiceType)
	assert.Equal(t, "Cow Level Crush", card.Services[1].Name)
	assert.Equal(t, "A great service", card.Services[1].Description)
}

func TestServiceToProviderCardResponse_EmptyServices(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svcService, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	provider := testProfile(testProviderID)

	card := svcService.ToProviderCardResponse(provider, []*models.Service{})

	assert.NotNil(t, card.Provider)
	assert.Equal(t, testProviderID, card.Provider.ID)
	assert.Empty(t, card.Services)
	assert.NotNil(t, card.Services, "services slice should be initialized, not nil")
}

// ---------------------------------------------------------------------------
// ListProviders
// ---------------------------------------------------------------------------

func TestServiceListProviders_Success(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svcService, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	ctx := context.Background()

	provider1 := testProfile("provider-aaa")
	provider2 := testProfile("provider-bbb")

	svc1 := testServiceModel("svc-1", "provider-aaa")
	svc2 := testServiceModel("svc-2", "provider-aaa")
	svc2.ServiceType = "crush"
	svc2.Name = "Crush Service"

	svc3 := testServiceModel("svc-3", "provider-bbb")
	svc3.ServiceType = "grush"
	svc3.Name = "Glitch Rush"

	providerWithServices := []repository.ProviderWithServices{
		{
			Provider: provider1,
			Services: []*models.Service{svc1, svc2},
		},
		{
			Provider: provider2,
			Services: []*models.Service{svc3},
		},
	}

	filter := repository.ServiceProviderFilter{
		Game:   "diablo2",
		Offset: 0,
		Limit:  10,
	}

	serviceRepo.On("ListProviders", mock.Anything, filter).Return(providerWithServices, 2, nil)

	results, count, err := svcService.ListProviders(ctx, filter)

	assert.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.Len(t, results, 2)

	// First provider card
	assert.Equal(t, "provider-aaa", results[0].Provider.ID)
	assert.Len(t, results[0].Services, 2)
	assert.Equal(t, "svc-1", results[0].Services[0].ID)
	assert.Equal(t, "svc-2", results[0].Services[1].ID)

	// Second provider card
	assert.Equal(t, "provider-bbb", results[1].Provider.ID)
	assert.Len(t, results[1].Services, 1)
	assert.Equal(t, "svc-3", results[1].Services[0].ID)
	assert.Equal(t, "grush", results[1].Services[0].ServiceType)

	serviceRepo.AssertExpectations(t)
}

func TestServiceListProviders_Empty(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svcService, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	ctx := context.Background()

	filter := repository.ServiceProviderFilter{
		Game:   "diablo2",
		Offset: 0,
		Limit:  10,
	}

	serviceRepo.On("ListProviders", mock.Anything, filter).Return([]repository.ProviderWithServices{}, 0, nil)

	results, count, err := svcService.ListProviders(ctx, filter)

	assert.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Empty(t, results)

	serviceRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// ListMyServices
// ---------------------------------------------------------------------------

func TestServiceListMyServices_Success(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svcService, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	ctx := context.Background()

	svc1 := testServiceModel("svc-1", testProviderID)
	svc2 := testServiceModel("svc-2", testProviderID)

	serviceRepo.On("ListByProviderID", mock.Anything, testProviderID, 0, 10).
		Return([]*models.Service{svc1, svc2}, 2, nil)

	results, count, err := svcService.ListMyServices(ctx, testProviderID, 0, 10)

	assert.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.Len(t, results, 2)
	assert.Equal(t, "svc-1", results[0].ID)
	assert.Equal(t, "svc-2", results[1].ID)

	serviceRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Pause
// ---------------------------------------------------------------------------

func TestServicePause_Success(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svc, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	ctx := context.Background()

	existing := testServiceModel(testServiceID, testProviderID)
	assert.Equal(t, "active", existing.Status, "precondition: service starts as active")

	serviceRepo.On("GetByID", mock.Anything, testServiceID).Return(existing, nil)
	serviceRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Service")).Return(nil)

	err := svc.Pause(ctx, testServiceID, testProviderID)

	assert.NoError(t, err)
	assert.Equal(t, "paused", existing.Status, "status should be set to paused")

	serviceRepo.AssertExpectations(t)
}

func TestServicePause_NotOwner(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svc, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	ctx := context.Background()
	otherUser := "other-user-999"

	existing := testServiceModel(testServiceID, testProviderID)
	serviceRepo.On("GetByID", mock.Anything, testServiceID).Return(existing, nil)

	err := svc.Pause(ctx, testServiceID, otherUser)

	assert.ErrorIs(t, err, ErrForbidden)
	assert.Equal(t, "active", existing.Status, "status should remain unchanged")

	serviceRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	serviceRepo.AssertExpectations(t)
}

func TestServicePause_AlreadyPaused(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svc, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	ctx := context.Background()

	existing := testServiceModel(testServiceID, testProviderID, withServiceStatus("paused"))
	serviceRepo.On("GetByID", mock.Anything, testServiceID).Return(existing, nil)

	err := svc.Pause(ctx, testServiceID, testProviderID)

	assert.ErrorIs(t, err, ErrInvalidState)

	serviceRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	serviceRepo.AssertExpectations(t)
}

func TestServicePause_Cancelled(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svc, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	ctx := context.Background()

	existing := testServiceModel(testServiceID, testProviderID, withServiceStatus("cancelled"))
	serviceRepo.On("GetByID", mock.Anything, testServiceID).Return(existing, nil)

	err := svc.Pause(ctx, testServiceID, testProviderID)

	assert.ErrorIs(t, err, ErrInvalidState)

	serviceRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	serviceRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Resume
// ---------------------------------------------------------------------------

func TestServiceResume_Success(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svc, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	ctx := context.Background()

	existing := testServiceModel(testServiceID, testProviderID, withServiceStatus("paused"))
	assert.Equal(t, "paused", existing.Status, "precondition: service starts as paused")

	serviceRepo.On("GetByID", mock.Anything, testServiceID).Return(existing, nil)
	serviceRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Service")).Return(nil)
	profileRepo.On("GetByID", mock.Anything, testProviderID).Return(testProfile(testProviderID), nil)

	err := svc.Resume(ctx, testServiceID, testProviderID)

	assert.NoError(t, err)
	assert.Equal(t, "active", existing.Status, "status should be set to active")

	serviceRepo.AssertExpectations(t)
}

func TestServiceResume_NotOwner(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svc, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	ctx := context.Background()
	otherUser := "other-user-999"

	existing := testServiceModel(testServiceID, testProviderID, withServiceStatus("paused"))
	serviceRepo.On("GetByID", mock.Anything, testServiceID).Return(existing, nil)

	err := svc.Resume(ctx, testServiceID, otherUser)

	assert.ErrorIs(t, err, ErrForbidden)
	assert.Equal(t, "paused", existing.Status, "status should remain unchanged")

	serviceRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	serviceRepo.AssertExpectations(t)
}

func TestServiceResume_AlreadyActive(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svc, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	ctx := context.Background()

	existing := testServiceModel(testServiceID, testProviderID) // default: active
	serviceRepo.On("GetByID", mock.Anything, testServiceID).Return(existing, nil)

	err := svc.Resume(ctx, testServiceID, testProviderID)

	assert.ErrorIs(t, err, ErrInvalidState)

	serviceRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	serviceRepo.AssertExpectations(t)
}

func TestServiceResume_Cancelled(t *testing.T) {
	profileRepo := new(mocks.MockProfileRepository)
	serviceRepo := new(mocks.MockServiceRepository)
	svc, _ := setupServiceService(profileRepo, serviceRepo, newTestRedis())

	ctx := context.Background()

	existing := testServiceModel(testServiceID, testProviderID, withServiceStatus("cancelled"))
	serviceRepo.On("GetByID", mock.Anything, testServiceID).Return(existing, nil)

	err := svc.Resume(ctx, testServiceID, testProviderID)

	assert.ErrorIs(t, err, ErrInvalidState)

	serviceRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	serviceRepo.AssertExpectations(t)
}
