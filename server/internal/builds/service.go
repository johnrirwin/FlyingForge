package builds

import (
	"context"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/johnrirwin/flyingforge/internal/database"
	"github.com/johnrirwin/flyingforge/internal/images"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

const (
	tempBuildTTL         = 24 * time.Hour
	defaultBuildTitle    = "Untitled Build"
	defaultTempBuildName = "Temporary Build"
)

// ServiceError represents a build service validation/runtime error.
type ServiceError struct {
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}

// ValidationError is returned when publish validation fails.
type ValidationError struct {
	Validation models.BuildValidationResult
}

func (e *ValidationError) Error() string {
	return "build publish validation failed"
}

type buildStore interface {
	Create(ctx context.Context, ownerUserID string, status models.BuildStatus, title string, description string, youtubeURL string, flightYouTubeURL string, sourceAircraftID string, token string, expiresAt *time.Time, parts []models.BuildPartInput) (*models.Build, error)
	ListByOwner(ctx context.Context, ownerUserID string, params models.BuildListParams) (*models.BuildListResponse, error)
	ListPublic(ctx context.Context, params models.BuildListParams, viewerUserID string) (*models.BuildListResponse, error)
	ListPublishedByOwner(ctx context.Context, ownerUserID string, viewerUserID string, limit int) ([]models.Build, error)
	ListForModeration(ctx context.Context, params models.BuildModerationListParams) (*models.BuildListResponse, error)
	GetByID(ctx context.Context, id string) (*models.Build, error)
	GetForOwner(ctx context.Context, id string, ownerUserID string) (*models.Build, error)
	GetPublic(ctx context.Context, id string, viewerUserID string) (*models.Build, error)
	GetTempByToken(ctx context.Context, token string) (*models.Build, error)
	GetForModeration(ctx context.Context, id string) (*models.Build, error)
	Update(ctx context.Context, id string, ownerUserID string, params models.UpdateBuildParams) (*models.Build, error)
	UpdateTempByToken(ctx context.Context, token string, params models.UpdateBuildParams, nextToken string) (*models.Build, error)
	UpdateForModeration(ctx context.Context, id string, params models.UpdateBuildParams) (*models.Build, error)
	SetStatus(ctx context.Context, id string, ownerUserID string, status models.BuildStatus) (*models.Build, error)
	SetImage(ctx context.Context, id string, ownerUserID string, imageAssetID string) (string, error)
	SetImageForModeration(ctx context.Context, id string, imageAssetID string) (string, error)
	GetImageForOwner(ctx context.Context, id string, ownerUserID string) ([]byte, error)
	GetPublicImage(ctx context.Context, id string) ([]byte, error)
	SetReaction(ctx context.Context, id string, userID string, reaction models.BuildReaction) (*models.Build, error)
	ClearReaction(ctx context.Context, id string, userID string) (*models.Build, error)
	GetImageForModeration(ctx context.Context, id string) ([]byte, error)
	DeleteImage(ctx context.Context, id string, ownerUserID string) (string, error)
	DeleteImageForModeration(ctx context.Context, id string) (string, error)
	ApproveForModeration(ctx context.Context, id string) (*models.Build, error)
	Delete(ctx context.Context, id string, ownerUserID string) (bool, error)
	DeleteExpiredTemp(ctx context.Context, cutoff time.Time) (int64, error)
}

type aircraftDetailsReader interface {
	GetDetails(ctx context.Context, id string, userID string) (*models.AircraftDetailsResponse, error)
	GetImage(ctx context.Context, id string, userID string) ([]byte, string, error)
}

type gearCatalogMigrator interface {
	MigrateInventoryItem(
		ctx context.Context,
		inventoryItemID, userID, name, manufacturer string,
		category models.EquipmentCategory,
		specs json.RawMessage,
	) (*models.GearCatalogItem, error)
}

type imagePipeline interface {
	ModerateAndPersist(ctx context.Context, req images.SaveRequest) (*models.ModerationDecision, *models.ImageAsset, error)
	PersistApprovedUpload(ctx context.Context, ownerUserID, uploadID string, entityType models.ImageEntityType, entityID string) (*models.ImageAsset, error)
	Delete(ctx context.Context, imageID string) error
}

// Service coordinates build business logic.
type Service struct {
	store         buildStore
	aircraftStore aircraftDetailsReader
	gearCatalog   gearCatalogMigrator
	imageSvc      imagePipeline
	logger        *logging.Logger
}

// NewService creates a build service.
func NewService(store *database.BuildStore, aircraftStore *database.AircraftStore, gearCatalogStore *database.GearCatalogStore, imageSvc *images.Service, logger *logging.Logger) *Service {
	return &Service{
		store:         store,
		aircraftStore: aircraftStore,
		gearCatalog:   gearCatalogStore,
		imageSvc:      imageSvc,
		logger:        logger,
	}
}

// NewServiceWithDeps is exposed for testing.
func NewServiceWithDeps(store buildStore, aircraftStore aircraftDetailsReader, gearCatalog gearCatalogMigrator, logger *logging.Logger) *Service {
	return &Service{
		store:         store,
		aircraftStore: aircraftStore,
		gearCatalog:   gearCatalog,
		imageSvc:      nil,
		logger:        logger,
	}
}

// ListPublic returns published builds.
func (s *Service) ListPublic(ctx context.Context, viewerUserID string, params models.BuildListParams) (*models.BuildListResponse, error) {
	resp, err := s.store.ListPublic(ctx, params, strings.TrimSpace(viewerUserID))
	if err != nil {
		return nil, err
	}
	for i := range resp.Builds {
		resp.Builds[i].Verified = isBuildVerified(&resp.Builds[i])
	}
	return resp, nil
}

// ListPublishedByOwner returns published builds for a specific pilot.
func (s *Service) ListPublishedByOwner(ctx context.Context, ownerUserID string, viewerUserID string, limit int) ([]models.Build, error) {
	builds, err := s.store.ListPublishedByOwner(ctx, strings.TrimSpace(ownerUserID), strings.TrimSpace(viewerUserID), limit)
	if err != nil {
		return nil, err
	}
	for i := range builds {
		builds[i].Verified = isBuildVerified(&builds[i])
	}
	return builds, nil
}

// ListForModeration returns builds queued for content moderation.
func (s *Service) ListForModeration(ctx context.Context, params models.BuildModerationListParams) (*models.BuildListResponse, error) {
	resp, err := s.store.ListForModeration(ctx, params)
	if err != nil {
		return nil, err
	}
	for i := range resp.Builds {
		resp.Builds[i].Verified = isBuildVerified(&resp.Builds[i])
	}
	return resp, nil
}

// GetPublic fetches one published build.
func (s *Service) GetPublic(ctx context.Context, id string, viewerUserID string) (*models.Build, error) {
	build, err := s.store.GetPublic(ctx, id, strings.TrimSpace(viewerUserID))
	if err != nil {
		return nil, err
	}
	if build == nil {
		return nil, nil
	}
	build.Verified = isBuildVerified(build)
	return build, nil
}

// SetReaction applies a like/dislike reaction from an authenticated user.
func (s *Service) SetReaction(ctx context.Context, id string, userID string, reaction models.BuildReaction) (*models.Build, error) {
	id = strings.TrimSpace(id)
	userID = strings.TrimSpace(userID)
	reaction = models.NormalizeBuildReaction(reaction)

	if id == "" {
		return nil, &ServiceError{Message: "build id is required"}
	}
	if userID == "" {
		return nil, &ServiceError{Message: "user id is required"}
	}
	if reaction != models.BuildReactionLike && reaction != models.BuildReactionDislike {
		return nil, &ServiceError{Message: "reaction must be LIKE or DISLIKE"}
	}

	build, err := s.store.SetReaction(ctx, id, userID, reaction)
	if err != nil {
		return nil, err
	}
	if build == nil {
		return nil, nil
	}
	build.Verified = isBuildVerified(build)
	return build, nil
}

// ClearReaction removes an authenticated user's build reaction.
func (s *Service) ClearReaction(ctx context.Context, id string, userID string) (*models.Build, error) {
	id = strings.TrimSpace(id)
	userID = strings.TrimSpace(userID)

	if id == "" {
		return nil, &ServiceError{Message: "build id is required"}
	}
	if userID == "" {
		return nil, &ServiceError{Message: "user id is required"}
	}

	build, err := s.store.ClearReaction(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if build == nil {
		return nil, nil
	}
	build.Verified = isBuildVerified(build)
	return build, nil
}

// CreateTemp creates a temporary anonymous build and returns share metadata.
func (s *Service) CreateTemp(ctx context.Context, ownerUserID string, params models.CreateBuildParams) (*models.TempBuildCreateResponse, error) {
	parts := normalizeParts(params.Parts)
	title := strings.TrimSpace(params.Title)
	if title == "" {
		title = defaultTempBuildName
	}
	description := strings.TrimSpace(params.Description)
	youtubeURL := strings.TrimSpace(params.YouTubeURL)
	flightYouTubeURL := strings.TrimSpace(params.FlightYouTubeURL)

	token, err := generateTempToken()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().UTC().Add(tempBuildTTL)

	build, err := s.store.Create(
		ctx,
		ownerUserID,
		models.BuildStatusTemp,
		title,
		description,
		youtubeURL,
		flightYouTubeURL,
		"",
		token,
		&expiresAt,
		parts,
	)
	if err != nil {
		return nil, err
	}
	build.Verified = isBuildVerified(build)
	build.Token = ""

	return &models.TempBuildCreateResponse{
		Build: build,
		Token: token,
		URL:   "/builds/temp/" + token,
	}, nil
}

// GetTempByToken returns a temp build if token is valid.
func (s *Service) GetTempByToken(ctx context.Context, token string) (*models.Build, error) {
	build, err := s.store.GetTempByToken(ctx, strings.TrimSpace(token))
	if err != nil {
		return nil, err
	}
	if build == nil {
		return nil, nil
	}
	build.Verified = isBuildVerified(build)
	build.Token = ""
	return build, nil
}

// UpdateTempByToken updates editable temp fields and rotates the temporary URL token.
func (s *Service) UpdateTempByToken(ctx context.Context, token string, params models.UpdateBuildParams) (*models.TempBuildCreateResponse, error) {
	if params.Title != nil {
		title := strings.TrimSpace(*params.Title)
		params.Title = &title
	}
	if params.Description != nil {
		desc := strings.TrimSpace(*params.Description)
		params.Description = &desc
	}
	if params.YouTubeURL != nil {
		url := strings.TrimSpace(*params.YouTubeURL)
		params.YouTubeURL = &url
	}
	if params.FlightYouTubeURL != nil {
		url := strings.TrimSpace(*params.FlightYouTubeURL)
		params.FlightYouTubeURL = &url
	}
	if params.Parts != nil {
		params.Parts = normalizeParts(params.Parts)
	}

	nextToken, err := generateTempToken()
	if err != nil {
		return nil, err
	}

	build, err := s.store.UpdateTempByToken(ctx, strings.TrimSpace(token), params, nextToken)
	if err != nil {
		return nil, err
	}
	if build == nil {
		return nil, nil
	}
	build.Verified = isBuildVerified(build)
	build.Token = ""
	return &models.TempBuildCreateResponse{
		Build: build,
		Token: nextToken,
		URL:   "/builds/temp/" + nextToken,
	}, nil
}

// ShareTempByToken snapshots a temporary/shared build into a new permanent shared link.
func (s *Service) ShareTempByToken(ctx context.Context, token string) (*models.TempBuildCreateResponse, error) {
	source, err := s.store.GetTempByToken(ctx, strings.TrimSpace(token))
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, nil
	}

	sharedToken, err := generateTempToken()
	if err != nil {
		return nil, err
	}

	sharedBuild, err := s.store.Create(
		ctx,
		source.OwnerUserID,
		models.BuildStatusShared,
		strings.TrimSpace(source.Title),
		strings.TrimSpace(source.Description),
		strings.TrimSpace(source.YouTubeURL),
		strings.TrimSpace(source.FlightYouTubeURL),
		strings.TrimSpace(source.SourceAircraftID),
		sharedToken,
		nil,
		normalizeParts(models.BuildPartInputsFromParts(source.Parts)),
	)
	if err != nil {
		return nil, err
	}

	sharedBuild.Verified = isBuildVerified(sharedBuild)
	sharedBuild.Token = ""

	return &models.TempBuildCreateResponse{
		Build: sharedBuild,
		Token: sharedToken,
		URL:   "/builds/temp/" + sharedToken,
	}, nil
}

// ListByOwner returns authenticated user's builds.
func (s *Service) ListByOwner(ctx context.Context, ownerUserID string, params models.BuildListParams) (*models.BuildListResponse, error) {
	resp, err := s.store.ListByOwner(ctx, ownerUserID, params)
	if err != nil {
		return nil, err
	}
	for i := range resp.Builds {
		resp.Builds[i].Verified = isBuildVerified(&resp.Builds[i])
	}
	return resp, nil
}

// CreateDraft creates a new draft build for a user.
func (s *Service) CreateDraft(ctx context.Context, ownerUserID string, params models.CreateBuildParams) (*models.Build, error) {
	title := strings.TrimSpace(params.Title)
	if title == "" {
		title = defaultBuildTitle
	}
	build, err := s.store.Create(
		ctx,
		ownerUserID,
		models.BuildStatusDraft,
		title,
		strings.TrimSpace(params.Description),
		strings.TrimSpace(params.YouTubeURL),
		strings.TrimSpace(params.FlightYouTubeURL),
		strings.TrimSpace(params.SourceAircraftID),
		"",
		nil,
		normalizeParts(params.Parts),
	)
	if err != nil {
		return nil, err
	}

	build.Verified = isBuildVerified(build)
	return build, nil
}

// CreateDraftFromAircraft creates a draft build pre-filled from aircraft components.
func (s *Service) CreateDraftFromAircraft(ctx context.Context, ownerUserID string, aircraftID string) (*models.Build, error) {
	aircraftID = strings.TrimSpace(aircraftID)
	if aircraftID == "" {
		return nil, &ServiceError{Message: "aircraft id is required"}
	}
	if s.aircraftStore == nil {
		return nil, &ServiceError{Message: "aircraft service unavailable"}
	}

	details, err := s.aircraftStore.GetDetails(ctx, aircraftID, ownerUserID)
	if err != nil {
		return nil, err
	}
	if details == nil || details.Aircraft.ID == "" {
		return nil, nil
	}

	parts := make([]models.BuildPartInput, 0)
	for _, component := range details.Components {
		if component.InventoryItem == nil {
			continue
		}
		gearType := aircraftComponentToGearType(component.Category)
		if gearType == "" {
			continue
		}
		catalogID := strings.TrimSpace(component.InventoryItem.CatalogID)
		if catalogID == "" && s.gearCatalog != nil {
			category := component.InventoryItem.Category
			if category == "" {
				category = componentCategoryToEquipmentCategory(component.Category)
			}
			if category != "" {
				catalogItem, err := s.gearCatalog.MigrateInventoryItem(
					ctx,
					component.InventoryItem.ID,
					ownerUserID,
					component.InventoryItem.Name,
					component.InventoryItem.Manufacturer,
					category,
					component.InventoryItem.Specs,
				)
				if err != nil {
					s.logger.Warn("Failed to backfill catalog entry while creating build from aircraft",
						logging.WithFields(map[string]interface{}{
							"aircraft_id": details.Aircraft.ID,
							"component":   component.Category,
							"error":       err.Error(),
						}))
				} else if catalogItem != nil {
					catalogID = strings.TrimSpace(catalogItem.ID)
				}
			}
		}
		if catalogID == "" {
			continue
		}
		parts = append(parts, models.BuildPartInput{
			GearType:      gearType,
			CatalogItemID: catalogID,
		})
	}

	title := strings.TrimSpace(details.Aircraft.Name)
	if title == "" {
		title = defaultBuildTitle
	}
	if !strings.Contains(strings.ToLower(title), "build") {
		title += " Build"
	}

	build, err := s.store.Create(
		ctx,
		ownerUserID,
		models.BuildStatusDraft,
		title,
		"",
		"",
		"",
		details.Aircraft.ID,
		"",
		nil,
		normalizeParts(parts),
	)
	if err != nil {
		return nil, err
	}

	if copied, err := s.copyAircraftImageToBuild(ctx, ownerUserID, details.Aircraft.ID, build.ID); err != nil {
		s.logger.Warn("Failed to copy aircraft image to new build",
			logging.WithFields(map[string]interface{}{
				"aircraft_id": details.Aircraft.ID,
				"build_id":    build.ID,
				"error":       err.Error(),
			}))
	} else if copied {
		if refreshed, refreshErr := s.store.GetForOwner(ctx, build.ID, ownerUserID); refreshErr == nil && refreshed != nil {
			build = refreshed
		}
	}
	build.Verified = isBuildVerified(build)
	return build, nil
}

// GetByOwner fetches one build for owner.
func (s *Service) GetByOwner(ctx context.Context, id string, ownerUserID string) (*models.Build, error) {
	build, err := s.store.GetForOwner(ctx, strings.TrimSpace(id), ownerUserID)
	if err != nil {
		return nil, err
	}
	if build == nil {
		return nil, nil
	}
	build.Verified = isBuildVerified(build)
	return build, nil
}

// GetForModeration fetches one build for content moderation workflows.
func (s *Service) GetForModeration(ctx context.Context, id string) (*models.Build, error) {
	build, err := s.store.GetForModeration(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if build == nil {
		return nil, nil
	}
	build.Verified = isBuildVerified(build)
	return build, nil
}

// UpdateByOwner updates an owned build.
func (s *Service) UpdateByOwner(ctx context.Context, id string, ownerUserID string, params models.UpdateBuildParams) (*models.Build, error) {
	if params.Title != nil {
		title := strings.TrimSpace(*params.Title)
		params.Title = &title
	}
	if params.Description != nil {
		desc := strings.TrimSpace(*params.Description)
		params.Description = &desc
	}
	if params.YouTubeURL != nil {
		url := strings.TrimSpace(*params.YouTubeURL)
		params.YouTubeURL = &url
	}
	if params.FlightYouTubeURL != nil {
		url := strings.TrimSpace(*params.FlightYouTubeURL)
		params.FlightYouTubeURL = &url
	}
	if params.Parts != nil {
		params.Parts = normalizeParts(params.Parts)
	}

	build, err := s.store.Update(ctx, strings.TrimSpace(id), ownerUserID, params)
	if err != nil {
		return nil, err
	}
	if build == nil {
		return nil, nil
	}
	build.Verified = isBuildVerified(build)
	return build, nil
}

// UpdateForModeration updates a build from the content moderation queue.
func (s *Service) UpdateForModeration(ctx context.Context, id string, params models.UpdateBuildParams) (*models.Build, error) {
	if params.Title != nil {
		title := strings.TrimSpace(*params.Title)
		params.Title = &title
	}
	if params.Description != nil {
		desc := strings.TrimSpace(*params.Description)
		params.Description = &desc
	}
	if params.YouTubeURL != nil {
		url := strings.TrimSpace(*params.YouTubeURL)
		params.YouTubeURL = &url
	}
	if params.FlightYouTubeURL != nil {
		url := strings.TrimSpace(*params.FlightYouTubeURL)
		params.FlightYouTubeURL = &url
	}
	if params.Parts != nil {
		params.Parts = normalizeParts(params.Parts)
	}

	build, err := s.store.UpdateForModeration(ctx, strings.TrimSpace(id), params)
	if err != nil {
		return nil, err
	}
	if build == nil {
		return nil, nil
	}
	build.Verified = isBuildVerified(build)
	return build, nil
}

// Publish validates and submits a build for content moderation.
func (s *Service) Publish(ctx context.Context, id string, ownerUserID string) (*models.Build, models.BuildValidationResult, error) {
	build, err := s.store.GetForOwner(ctx, strings.TrimSpace(id), ownerUserID)
	if err != nil {
		return nil, models.BuildValidationResult{}, err
	}
	if build == nil {
		return nil, models.BuildValidationResult{}, nil
	}

	validation := ValidateForPublish(build)
	if !validation.Valid {
		return nil, validation, &ValidationError{Validation: validation}
	}

	updated, err := s.store.SetStatus(ctx, build.ID, ownerUserID, models.BuildStatusPendingReview)
	if err != nil {
		return nil, validation, err
	}
	if updated == nil {
		return nil, validation, nil
	}
	updated.Verified = isBuildVerified(updated)
	return updated, validation, nil
}

// Unpublish removes a build from public listings.
func (s *Service) Unpublish(ctx context.Context, id string, ownerUserID string) (*models.Build, error) {
	updated, err := s.store.SetStatus(ctx, strings.TrimSpace(id), ownerUserID, models.BuildStatusUnpublished)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, nil
	}
	updated.Verified = isBuildVerified(updated)
	return updated, nil
}

// UnpublishForModeration removes a build from public listings via content moderation tools.
func (s *Service) UnpublishForModeration(ctx context.Context, id string) (*models.Build, error) {
	build, err := s.store.GetForModeration(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if build == nil {
		return nil, nil
	}
	if build.Status != models.BuildStatusPublished && build.Status != models.BuildStatusPendingReview {
		return nil, &ServiceError{Message: "build is not published"}
	}
	if strings.TrimSpace(build.OwnerUserID) == "" {
		return nil, &ServiceError{Message: "build owner missing"}
	}

	updated, err := s.store.SetStatus(ctx, build.ID, build.OwnerUserID, models.BuildStatusUnpublished)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, nil
	}
	updated.Verified = isBuildVerified(updated)
	return updated, nil
}

// ApproveForModeration publishes a pending build from the moderation queue.
func (s *Service) ApproveForModeration(ctx context.Context, id string) (*models.Build, models.BuildValidationResult, error) {
	build, err := s.store.GetForModeration(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, models.BuildValidationResult{}, err
	}
	if build == nil {
		return nil, models.BuildValidationResult{}, nil
	}

	validation := ValidateForPublish(build)
	if !validation.Valid {
		return nil, validation, &ValidationError{Validation: validation}
	}
	if build.Status != models.BuildStatusPendingReview {
		return nil, validation, &ServiceError{Message: "build is not pending moderation"}
	}

	updated, err := s.store.ApproveForModeration(ctx, build.ID)
	if err != nil {
		return nil, validation, err
	}
	if updated == nil {
		return nil, validation, nil
	}
	updated.Verified = isBuildVerified(updated)
	return updated, validation, nil
}

// DeleteByOwner deletes an owned non-temp build that is not currently published or pending review.
func (s *Service) DeleteByOwner(ctx context.Context, id string, ownerUserID string) (bool, error) {
	build, err := s.store.GetForOwner(ctx, strings.TrimSpace(id), ownerUserID)
	if err != nil {
		return false, err
	}
	if build == nil {
		return false, nil
	}
	if build.Status == models.BuildStatusPublished || build.Status == models.BuildStatusPendingReview {
		return false, &ServiceError{Message: "published builds must be unpublished before deletion"}
	}

	return s.store.Delete(ctx, strings.TrimSpace(id), ownerUserID)
}

// SetImage uploads an image for a build.
func (s *Service) SetImage(ctx context.Context, userID string, params models.SetBuildImageParams) (*models.ModerationDecision, error) {
	if strings.TrimSpace(params.BuildID) == "" {
		return nil, &ServiceError{Message: "build id is required"}
	}
	if s.imageSvc == nil {
		return nil, &ServiceError{Message: "image moderation unavailable"}
	}

	build, err := s.store.GetForOwner(ctx, strings.TrimSpace(params.BuildID), userID)
	if err != nil {
		return nil, err
	}
	if build == nil {
		return nil, &ServiceError{Message: "build not found"}
	}

	var (
		decision *models.ModerationDecision
		asset    *models.ImageAsset
	)

	uploadID := strings.TrimSpace(params.UploadID)
	if uploadID != "" {
		asset, err = s.imageSvc.PersistApprovedUpload(ctx, userID, uploadID, models.ImageEntityBuild, build.ID)
		if err != nil {
			return nil, err
		}
		decision = &models.ModerationDecision{
			Status: models.ImageModerationApproved,
			Reason: "Approved",
		}
		if params.ImageType == "" {
			params.ImageType = http.DetectContentType(asset.ImageBytes)
		}
	} else {
		if len(params.ImageData) == 0 {
			return nil, &ServiceError{Message: "image data is required"}
		}
		if params.ImageType != "image/jpeg" && params.ImageType != "image/png" {
			return nil, &ServiceError{Message: "image must be JPEG or PNG"}
		}

		const maxImageSize = 2 * 1024 * 1024
		if len(params.ImageData) > maxImageSize {
			return nil, &ServiceError{Message: "image must be less than 2MB"}
		}

		decision, asset, err = s.imageSvc.ModerateAndPersist(ctx, images.SaveRequest{
			OwnerUserID: userID,
			EntityType:  models.ImageEntityBuild,
			EntityID:    build.ID,
			ImageBytes:  params.ImageData,
		})
		if err != nil {
			return nil, err
		}
		if decision == nil || decision.Status != models.ImageModerationApproved {
			return decision, nil
		}
	}

	if asset == nil {
		return nil, &ServiceError{Message: "failed to persist build image"}
	}

	previousAssetID, err := s.store.SetImage(ctx, build.ID, userID, asset.ID)
	if err != nil {
		_ = s.imageSvc.Delete(ctx, asset.ID)
		return nil, err
	}
	if previousAssetID != "" && previousAssetID != asset.ID {
		_ = s.imageSvc.Delete(ctx, previousAssetID)
	}

	return decision, nil
}

// SetImageForModeration uploads an image for moderator-curated build updates.
func (s *Service) SetImageForModeration(ctx context.Context, moderatorUserID string, params models.SetBuildImageParams) (*models.ModerationDecision, error) {
	if strings.TrimSpace(params.BuildID) == "" {
		return nil, &ServiceError{Message: "build id is required"}
	}
	if s.imageSvc == nil {
		return nil, &ServiceError{Message: "image moderation unavailable"}
	}

	build, err := s.store.GetForModeration(ctx, strings.TrimSpace(params.BuildID))
	if err != nil {
		return nil, err
	}
	if build == nil {
		return nil, &ServiceError{Message: "build not found"}
	}

	if len(params.ImageData) == 0 {
		return nil, &ServiceError{Message: "image data is required"}
	}
	if params.ImageType != "image/jpeg" && params.ImageType != "image/png" {
		return nil, &ServiceError{Message: "image must be JPEG or PNG"}
	}

	const maxImageSize = 2 * 1024 * 1024
	if len(params.ImageData) > maxImageSize {
		return nil, &ServiceError{Message: "image must be less than 2MB"}
	}

	decision, asset, err := s.imageSvc.ModerateAndPersist(ctx, images.SaveRequest{
		OwnerUserID: moderatorUserID,
		EntityType:  models.ImageEntityBuild,
		EntityID:    build.ID,
		ImageBytes:  params.ImageData,
	})
	if err != nil {
		return nil, err
	}
	if decision == nil || decision.Status != models.ImageModerationApproved || asset == nil {
		return decision, nil
	}

	previousAssetID, err := s.store.SetImageForModeration(ctx, build.ID, asset.ID)
	if err != nil {
		_ = s.imageSvc.Delete(ctx, asset.ID)
		return nil, err
	}
	if previousAssetID != "" && previousAssetID != asset.ID {
		_ = s.imageSvc.Delete(ctx, previousAssetID)
	}

	return decision, nil
}

// GetImage retrieves a build image for its owner.
func (s *Service) GetImage(ctx context.Context, buildID string, userID string) ([]byte, string, error) {
	imageData, err := s.store.GetImageForOwner(ctx, strings.TrimSpace(buildID), userID)
	if err != nil {
		return nil, "", err
	}
	if len(imageData) == 0 {
		return imageData, "", nil
	}
	return imageData, http.DetectContentType(imageData), nil
}

// GetPublicImage retrieves a published build image for public views.
func (s *Service) GetPublicImage(ctx context.Context, buildID string) ([]byte, string, error) {
	imageData, err := s.store.GetPublicImage(ctx, strings.TrimSpace(buildID))
	if err != nil {
		return nil, "", err
	}
	if len(imageData) == 0 {
		return imageData, "", nil
	}
	return imageData, http.DetectContentType(imageData), nil
}

// GetImageForModeration retrieves a build image for moderation views.
func (s *Service) GetImageForModeration(ctx context.Context, buildID string) ([]byte, string, error) {
	imageData, err := s.store.GetImageForModeration(ctx, strings.TrimSpace(buildID))
	if err != nil {
		return nil, "", err
	}
	if len(imageData) == 0 {
		return imageData, "", nil
	}
	return imageData, http.DetectContentType(imageData), nil
}

// DeleteImage removes an image from a build.
func (s *Service) DeleteImage(ctx context.Context, buildID string, userID string) error {
	build, err := s.store.GetForOwner(ctx, strings.TrimSpace(buildID), userID)
	if err != nil {
		return err
	}
	if build == nil {
		return &ServiceError{Message: "build not found"}
	}

	previousAssetID, err := s.store.DeleteImage(ctx, build.ID, userID)
	if err != nil {
		return err
	}
	if previousAssetID != "" && s.imageSvc != nil {
		_ = s.imageSvc.Delete(ctx, previousAssetID)
	}
	return nil
}

// DeleteImageForModeration removes an image from a build via content moderation.
func (s *Service) DeleteImageForModeration(ctx context.Context, buildID string) error {
	build, err := s.store.GetForModeration(ctx, strings.TrimSpace(buildID))
	if err != nil {
		return err
	}
	if build == nil {
		return &ServiceError{Message: "build not found"}
	}

	previousAssetID, err := s.store.DeleteImageForModeration(ctx, build.ID)
	if err != nil {
		return err
	}
	if previousAssetID != "" && s.imageSvc != nil {
		_ = s.imageSvc.Delete(ctx, previousAssetID)
	}
	return nil
}

// CleanupExpiredTemp deletes expired temp builds.
func (s *Service) CleanupExpiredTemp(ctx context.Context) (int64, error) {
	return s.store.DeleteExpiredTemp(ctx, time.Now().UTC())
}

func (s *Service) copyAircraftImageToBuild(ctx context.Context, userID string, aircraftID string, buildID string) (bool, error) {
	if s.aircraftStore == nil || s.imageSvc == nil {
		return false, nil
	}

	imageData, _, err := s.aircraftStore.GetImage(ctx, aircraftID, userID)
	if err != nil {
		return false, err
	}
	if len(imageData) == 0 {
		return false, nil
	}

	decision, asset, err := s.imageSvc.ModerateAndPersist(ctx, images.SaveRequest{
		OwnerUserID: userID,
		EntityType:  models.ImageEntityBuild,
		EntityID:    buildID,
		ImageBytes:  imageData,
	})
	if err != nil {
		return false, err
	}
	if decision == nil || decision.Status != models.ImageModerationApproved || asset == nil {
		return false, nil
	}

	previousAssetID, err := s.store.SetImage(ctx, buildID, userID, asset.ID)
	if err != nil {
		_ = s.imageSvc.Delete(ctx, asset.ID)
		return false, err
	}
	if previousAssetID != "" && previousAssetID != asset.ID {
		_ = s.imageSvc.Delete(ctx, previousAssetID)
	}

	return true, nil
}

// ValidateForPublish enforces public-eligibility rules.
func ValidateForPublish(build *models.Build) models.BuildValidationResult {
	if build == nil {
		return models.BuildValidationResult{
			Valid: false,
			Errors: []models.BuildValidationError{{
				Category: "build",
				Code:     "missing_build",
				Message:  "Build not found",
			}},
		}
	}

	errors := make([]models.BuildValidationError, 0)

	if strings.TrimSpace(build.Description) == "" {
		errors = append(errors, models.BuildValidationError{
			Category: "description",
			Code:     "missing_required",
			Message:  "Description is required before submitting for review",
		})
	}

	if strings.TrimSpace(build.ImageAssetID) == "" {
		errors = append(errors, models.BuildValidationError{
			Category: "image",
			Code:     "missing_required",
			Message:  "Build image is required before submitting for review",
		})
	}

	required := []struct {
		gearType models.GearType
		label    string
		category string
	}{
		{models.GearTypeFrame, "Frame", "frame"},
		{models.GearTypeMotor, "Motors", "motor"},
		{models.GearTypeReceiver, "Receiver", "receiver"},
		{models.GearTypeVTX, "VTX", "vtx"},
	}

	for _, req := range required {
		if !hasPart(build.Parts, req.gearType) {
			errors = append(errors, models.BuildValidationError{
				Category: req.category,
				Code:     "missing_required",
				Message:  req.label + " is required",
			})
		}
	}

	hasAIO := hasPart(build.Parts, models.GearTypeAIO)
	hasStack := hasPart(build.Parts, models.GearTypeStack)
	hasFC := hasPart(build.Parts, models.GearTypeFC)
	hasESC := hasPart(build.Parts, models.GearTypeESC)
	if !hasCompletePowerStack(build.Parts) {
		errors = append(errors, models.BuildValidationError{
			Category: "power-stack",
			Code:     "missing_required",
			Message:  "Power stack requires an AIO, an FC/ESC stack, or both FC and ESC",
		})
	}

	if build.SourceAircraftID != "" {
		checkPublished := []struct {
			gearType models.GearType
			label    string
			category string
		}{
			{models.GearTypeFrame, "Frame", "frame"},
			{models.GearTypeMotor, "Motors", "motor"},
			{models.GearTypeReceiver, "Receiver", "receiver"},
			{models.GearTypeVTX, "VTX", "vtx"},
		}
		if hasAIO {
			checkPublished = append(checkPublished, struct {
				gearType models.GearType
				label    string
				category string
			}{models.GearTypeAIO, "AIO", "aio"})
		}
		if hasStack {
			checkPublished = append(checkPublished, struct {
				gearType models.GearType
				label    string
				category string
			}{models.GearTypeStack, "FC/ESC Stack", "stack"})
		}
		if hasFC {
			checkPublished = append(checkPublished, struct {
				gearType models.GearType
				label    string
				category string
			}{models.GearTypeFC, "Flight Controller", "fc"})
		}
		if hasESC {
			checkPublished = append(checkPublished, struct {
				gearType models.GearType
				label    string
				category string
			}{models.GearTypeESC, "ESC", "esc"})
		}

		for _, part := range checkPublished {
			p := findFirstPart(build.Parts, part.gearType)
			if p == nil {
				continue
			}
			if p.CatalogItem == nil || models.NormalizeCatalogStatus(p.CatalogItem.Status) != models.CatalogStatusPublished {
				name := part.label
				if p.CatalogItem != nil {
					if displayName := p.CatalogItem.DisplayName(); displayName != "" {
						name = displayName
					}
				}
				errors = append(errors, models.BuildValidationError{
					Category: part.category,
					Code:     "not_published",
					Message:  fmt.Sprintf("%s is not a published catalog item", name),
				})
			}
		}
	}

	return models.BuildValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

func normalizeParts(parts []models.BuildPartInput) []models.BuildPartInput {
	if parts == nil {
		return nil
	}

	type key struct {
		gearType models.GearType
		position int
	}

	normalized := make(map[key]models.BuildPartInput)
	for _, part := range parts {
		part.GearType = models.GearType(strings.TrimSpace(string(part.GearType)))
		part.CatalogItemID = strings.TrimSpace(part.CatalogItemID)
		part.Notes = strings.TrimSpace(part.Notes)
		if part.GearType == "" || part.CatalogItemID == "" {
			continue
		}
		if part.Position < 0 {
			part.Position = 0
		}
		normalized[key{gearType: part.GearType, position: part.Position}] = part
	}

	result := make([]models.BuildPartInput, 0, len(normalized))
	for _, part := range normalized {
		result = append(result, part)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].GearType == result[j].GearType {
			return result[i].Position < result[j].Position
		}
		return result[i].GearType < result[j].GearType
	})
	return result
}

func hasPart(parts []models.BuildPart, gearType models.GearType) bool {
	for _, part := range parts {
		if part.GearType == gearType && strings.TrimSpace(part.CatalogItemID) != "" {
			return true
		}
	}
	return false
}

func hasCompletePowerStack(parts []models.BuildPart) bool {
	hasAIO := hasPart(parts, models.GearTypeAIO)
	hasStack := hasPart(parts, models.GearTypeStack)
	hasFC := hasPart(parts, models.GearTypeFC)
	hasESC := hasPart(parts, models.GearTypeESC)
	return hasAIO || hasStack || (hasFC && hasESC)
}

func hasRequiredCoreParts(parts []models.BuildPart) bool {
	required := []models.GearType{
		models.GearTypeFrame,
		models.GearTypeMotor,
		models.GearTypeReceiver,
		models.GearTypeVTX,
	}
	for _, gearType := range required {
		if !hasPart(parts, gearType) {
			return false
		}
	}
	return hasCompletePowerStack(parts)
}

func findFirstPart(parts []models.BuildPart, gearType models.GearType) *models.BuildPart {
	for i := range parts {
		if parts[i].GearType == gearType {
			return &parts[i]
		}
	}
	return nil
}

func isBuildVerified(build *models.Build) bool {
	if build == nil {
		return false
	}
	if len(build.Parts) == 0 {
		return false
	}
	if !hasRequiredCoreParts(build.Parts) {
		return false
	}
	for _, part := range build.Parts {
		if strings.TrimSpace(part.CatalogItemID) == "" {
			return false
		}
		if part.CatalogItem == nil {
			return false
		}
		if models.NormalizeCatalogStatus(part.CatalogItem.Status) != models.CatalogStatusPublished {
			return false
		}
	}
	return true
}

func aircraftComponentToGearType(category models.ComponentCategory) models.GearType {
	switch models.NormalizeComponentCategory(category) {
	case models.ComponentCategoryFrame:
		return models.GearTypeFrame
	case models.ComponentCategoryMotors:
		return models.GearTypeMotor
	case models.ComponentCategoryAIO:
		return models.GearTypeAIO
	case models.ComponentCategoryStack:
		return models.GearTypeStack
	case models.ComponentCategoryFC:
		return models.GearTypeFC
	case models.ComponentCategoryESC:
		return models.GearTypeESC
	case models.ComponentCategoryReceiver:
		return models.GearTypeReceiver
	case models.ComponentCategoryVTX:
		return models.GearTypeVTX
	case models.ComponentCategoryCamera:
		return models.GearTypeCamera
	case models.ComponentCategoryProps:
		return models.GearTypeProp
	case models.ComponentCategoryAntenna:
		return models.GearTypeAntenna
	case models.ComponentCategoryGPS:
		return models.GearTypeGPS
	default:
		return ""
	}
}

func componentCategoryToEquipmentCategory(category models.ComponentCategory) models.EquipmentCategory {
	switch models.NormalizeComponentCategory(category) {
	case models.ComponentCategoryFrame:
		return models.CategoryFrames
	case models.ComponentCategoryMotors:
		return models.CategoryMotors
	case models.ComponentCategoryAIO:
		return models.CategoryAIO
	case models.ComponentCategoryStack:
		return models.CategoryStacks
	case models.ComponentCategoryFC:
		return models.CategoryFC
	case models.ComponentCategoryESC:
		return models.CategoryESC
	case models.ComponentCategoryReceiver:
		return models.CategoryReceivers
	case models.ComponentCategoryVTX:
		return models.CategoryVTX
	case models.ComponentCategoryCamera:
		return models.CategoryCameras
	case models.ComponentCategoryProps:
		return models.CategoryPropellers
	case models.ComponentCategoryAntenna:
		return models.CategoryAntennas
	case models.ComponentCategoryGPS:
		return models.CategoryGPS
	default:
		return ""
	}
}

func generateTempToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := crand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to generate temp token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
