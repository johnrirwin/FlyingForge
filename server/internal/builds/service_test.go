package builds

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

func TestValidateForPublish_MissingRequiredCategories(t *testing.T) {
	build := &models.Build{
		Status: models.BuildStatusDraft,
		Parts: []models.BuildPart{
			{GearType: models.GearTypeFrame, CatalogItemID: "frame-1", CatalogItem: publishedCatalog("frame-1", models.GearTypeFrame)},
		},
	}

	result := ValidateForPublish(build)
	if result.Valid {
		t.Fatalf("expected validation to fail")
	}

	assertHasValidationCode(t, result.Errors, "motor", "missing_required")
	assertHasValidationCode(t, result.Errors, "receiver", "missing_required")
	assertHasValidationCode(t, result.Errors, "vtx", "missing_required")
	assertHasValidationCode(t, result.Errors, "power-stack", "missing_required")
}

func TestValidateForPublish_PowerStackLogic(t *testing.T) {
	tests := []struct {
		name      string
		parts     []models.BuildPart
		wantValid bool
	}{
		{
			name: "valid with aio",
			parts: []models.BuildPart{
				{GearType: models.GearTypeFrame, CatalogItemID: "frame-1", CatalogItem: publishedCatalog("frame-1", models.GearTypeFrame)},
				{GearType: models.GearTypeMotor, CatalogItemID: "motor-1", CatalogItem: publishedCatalog("motor-1", models.GearTypeMotor)},
				{GearType: models.GearTypeAIO, CatalogItemID: "aio-1", CatalogItem: publishedCatalog("aio-1", models.GearTypeAIO)},
				{GearType: models.GearTypeReceiver, CatalogItemID: "rx-1", CatalogItem: publishedCatalog("rx-1", models.GearTypeReceiver)},
				{GearType: models.GearTypeVTX, CatalogItemID: "vtx-1", CatalogItem: publishedCatalog("vtx-1", models.GearTypeVTX)},
			},
			wantValid: true,
		},
		{
			name: "invalid with only fc",
			parts: []models.BuildPart{
				{GearType: models.GearTypeFrame, CatalogItemID: "frame-1", CatalogItem: publishedCatalog("frame-1", models.GearTypeFrame)},
				{GearType: models.GearTypeMotor, CatalogItemID: "motor-1", CatalogItem: publishedCatalog("motor-1", models.GearTypeMotor)},
				{GearType: models.GearTypeFC, CatalogItemID: "fc-1", CatalogItem: publishedCatalog("fc-1", models.GearTypeFC)},
				{GearType: models.GearTypeReceiver, CatalogItemID: "rx-1", CatalogItem: publishedCatalog("rx-1", models.GearTypeReceiver)},
				{GearType: models.GearTypeVTX, CatalogItemID: "vtx-1", CatalogItem: publishedCatalog("vtx-1", models.GearTypeVTX)},
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateForPublish(&models.Build{Parts: tt.parts})
			if result.Valid != tt.wantValid {
				t.Fatalf("valid=%v want %v, errors=%v", result.Valid, tt.wantValid, result.Errors)
			}
		})
	}
}

func TestValidateForPublish_FromAircraftRequiresPublishedCatalogParts(t *testing.T) {
	build := &models.Build{
		SourceAircraftID: "aircraft-1",
		Parts: []models.BuildPart{
			{GearType: models.GearTypeFrame, CatalogItemID: "frame-1", CatalogItem: pendingCatalog("frame-1", models.GearTypeFrame)},
			{GearType: models.GearTypeMotor, CatalogItemID: "motor-1", CatalogItem: publishedCatalog("motor-1", models.GearTypeMotor)},
			{GearType: models.GearTypeAIO, CatalogItemID: "aio-1", CatalogItem: pendingCatalog("aio-1", models.GearTypeAIO)},
			{GearType: models.GearTypeReceiver, CatalogItemID: "rx-1", CatalogItem: publishedCatalog("rx-1", models.GearTypeReceiver)},
			{GearType: models.GearTypeVTX, CatalogItemID: "vtx-1", CatalogItem: pendingCatalog("vtx-1", models.GearTypeVTX)},
		},
	}

	result := ValidateForPublish(build)
	if result.Valid {
		t.Fatalf("expected validation to fail")
	}

	assertHasValidationCode(t, result.Errors, "frame", "not_published")
	assertHasValidationCode(t, result.Errors, "aio", "not_published")
	assertHasValidationCode(t, result.Errors, "vtx", "not_published")
}

func TestTempBuildCreateAndRetrieve(t *testing.T) {
	ctx := context.Background()
	store := newFakeBuildStore()
	svc := NewServiceWithDeps(store, nil, nil, logging.New(logging.LevelError))

	created, err := svc.CreateTemp(ctx, "", models.CreateBuildParams{
		Title: "Visitor Build",
		Parts: []models.BuildPartInput{{GearType: models.GearTypeFrame, CatalogItemID: "frame-1"}},
	})
	if err != nil {
		t.Fatalf("CreateTemp error: %v", err)
	}
	if created.Build == nil {
		t.Fatalf("expected build in response")
	}
	if created.Token == "" {
		t.Fatalf("expected token")
	}
	if !strings.Contains(created.URL, created.Token) {
		t.Fatalf("expected URL to contain token")
	}
	if created.Build.Status != models.BuildStatusTemp {
		t.Fatalf("status=%s want TEMP", created.Build.Status)
	}
	if created.Build.ExpiresAt == nil {
		t.Fatalf("expected expiresAt")
	}
	if ttl := created.Build.ExpiresAt.Sub(created.Build.CreatedAt); ttl < 23*time.Hour || ttl > 25*time.Hour {
		t.Fatalf("unexpected TTL: %v", ttl)
	}

	fetched, err := svc.GetTempByToken(ctx, created.Token)
	if err != nil {
		t.Fatalf("GetTempByToken error: %v", err)
	}
	if fetched == nil {
		t.Fatalf("expected fetched build")
	}
	if fetched.ID != created.Build.ID {
		t.Fatalf("id=%s want %s", fetched.ID, created.Build.ID)
	}
	if len(fetched.Parts) != 1 || fetched.Parts[0].CatalogItemID != "frame-1" {
		t.Fatalf("unexpected parts: %+v", fetched.Parts)
	}
}

func TestDeleteByOwner(t *testing.T) {
	ctx := context.Background()
	store := newFakeBuildStore()
	svc := NewServiceWithDeps(store, nil, nil, logging.New(logging.LevelError))

	created, err := svc.CreateDraft(ctx, "user-1", models.CreateBuildParams{Title: "To Delete"})
	if err != nil {
		t.Fatalf("CreateDraft error: %v", err)
	}

	deleted, err := svc.DeleteByOwner(ctx, created.ID, "user-1")
	if err != nil {
		t.Fatalf("DeleteByOwner error: %v", err)
	}
	if !deleted {
		t.Fatalf("expected build to be deleted")
	}

	fetched, err := svc.GetByOwner(ctx, created.ID, "user-1")
	if err != nil {
		t.Fatalf("GetByOwner error: %v", err)
	}
	if fetched != nil {
		t.Fatalf("expected build to be deleted, got %+v", fetched)
	}
}

func assertHasValidationCode(t *testing.T, errs []models.BuildValidationError, category, code string) {
	t.Helper()
	for _, err := range errs {
		if err.Category == category && err.Code == code {
			return
		}
	}
	t.Fatalf("expected error category=%s code=%s, got=%+v", category, code, errs)
}

func publishedCatalog(id string, gearType models.GearType) *models.BuildCatalogItem {
	return &models.BuildCatalogItem{ID: id, GearType: gearType, Brand: "Brand", Model: "Model", Status: models.CatalogStatusPublished}
}

func pendingCatalog(id string, gearType models.GearType) *models.BuildCatalogItem {
	return &models.BuildCatalogItem{ID: id, GearType: gearType, Brand: "Brand", Model: "Model", Status: models.CatalogStatusPending}
}

// fakeBuildStore is a lightweight in-memory store used for service tests.
type fakeBuildStore struct {
	byID    map[string]*models.Build
	byToken map[string]string
	nextID  int
}

func newFakeBuildStore() *fakeBuildStore {
	return &fakeBuildStore{
		byID:    map[string]*models.Build{},
		byToken: map[string]string{},
	}
}

func (s *fakeBuildStore) Create(ctx context.Context, ownerUserID string, status models.BuildStatus, title string, description string, sourceAircraftID string, token string, expiresAt *time.Time, parts []models.BuildPartInput) (*models.Build, error) {
	s.nextID++
	id := "build-" + strconvItoa(s.nextID)
	now := time.Now().UTC()
	build := &models.Build{
		ID:               id,
		OwnerUserID:      ownerUserID,
		Status:           status,
		Token:            token,
		ExpiresAt:        expiresAt,
		Title:            title,
		Description:      description,
		SourceAircraftID: sourceAircraftID,
		CreatedAt:        now,
		UpdatedAt:        now,
		Parts:            convertParts(parts),
	}
	s.byID[id] = cloneBuild(build)
	if token != "" {
		s.byToken[token] = id
	}
	return cloneBuild(build), nil
}

func (s *fakeBuildStore) ListByOwner(ctx context.Context, ownerUserID string, params models.BuildListParams) (*models.BuildListResponse, error) {
	items := make([]models.Build, 0)
	for _, build := range s.byID {
		if build.OwnerUserID == ownerUserID && build.Status != models.BuildStatusTemp {
			items = append(items, *cloneBuild(build))
		}
	}
	return &models.BuildListResponse{Builds: items, TotalCount: len(items)}, nil
}

func (s *fakeBuildStore) ListPublic(ctx context.Context, params models.BuildListParams) (*models.BuildListResponse, error) {
	items := make([]models.Build, 0)
	for _, build := range s.byID {
		if build.Status == models.BuildStatusPublished {
			items = append(items, *cloneBuild(build))
		}
	}
	return &models.BuildListResponse{Builds: items, TotalCount: len(items)}, nil
}

func (s *fakeBuildStore) GetByID(ctx context.Context, id string) (*models.Build, error) {
	build := s.byID[id]
	if build == nil {
		return nil, nil
	}
	return cloneBuild(build), nil
}

func (s *fakeBuildStore) GetForOwner(ctx context.Context, id string, ownerUserID string) (*models.Build, error) {
	build := s.byID[id]
	if build == nil || build.OwnerUserID != ownerUserID {
		return nil, nil
	}
	return cloneBuild(build), nil
}

func (s *fakeBuildStore) GetPublic(ctx context.Context, id string) (*models.Build, error) {
	build := s.byID[id]
	if build == nil || build.Status != models.BuildStatusPublished {
		return nil, nil
	}
	return cloneBuild(build), nil
}

func (s *fakeBuildStore) GetTempByToken(ctx context.Context, token string) (*models.Build, error) {
	id, ok := s.byToken[token]
	if !ok {
		return nil, nil
	}
	build := s.byID[id]
	if build == nil || build.Status != models.BuildStatusTemp {
		return nil, nil
	}
	if build.ExpiresAt != nil && build.ExpiresAt.Before(time.Now().UTC()) {
		return nil, nil
	}
	return cloneBuild(build), nil
}

func (s *fakeBuildStore) Update(ctx context.Context, id string, ownerUserID string, params models.UpdateBuildParams) (*models.Build, error) {
	build := s.byID[id]
	if build == nil || build.OwnerUserID != ownerUserID {
		return nil, nil
	}
	if params.Title != nil {
		build.Title = *params.Title
	}
	if params.Description != nil {
		build.Description = *params.Description
	}
	if params.Parts != nil {
		build.Parts = convertParts(params.Parts)
	}
	build.UpdatedAt = time.Now().UTC()
	return cloneBuild(build), nil
}

func (s *fakeBuildStore) UpdateTempByToken(ctx context.Context, token string, params models.UpdateBuildParams) (*models.Build, error) {
	id, ok := s.byToken[token]
	if !ok {
		return nil, nil
	}
	build := s.byID[id]
	if build == nil || build.Status != models.BuildStatusTemp {
		return nil, nil
	}
	if params.Title != nil {
		build.Title = *params.Title
	}
	if params.Description != nil {
		build.Description = *params.Description
	}
	if params.Parts != nil {
		build.Parts = convertParts(params.Parts)
	}
	build.UpdatedAt = time.Now().UTC()
	return cloneBuild(build), nil
}

func (s *fakeBuildStore) SetStatus(ctx context.Context, id string, ownerUserID string, status models.BuildStatus) (*models.Build, error) {
	build := s.byID[id]
	if build == nil || build.OwnerUserID != ownerUserID {
		return nil, nil
	}
	build.Status = status
	now := time.Now().UTC()
	if status == models.BuildStatusPublished {
		build.PublishedAt = &now
	} else if status == models.BuildStatusUnpublished {
		build.PublishedAt = nil
	}
	build.UpdatedAt = now
	return cloneBuild(build), nil
}

func (s *fakeBuildStore) Delete(ctx context.Context, id string, ownerUserID string) (bool, error) {
	build := s.byID[id]
	if build == nil || build.OwnerUserID != ownerUserID || build.Status == models.BuildStatusTemp {
		return false, nil
	}
	delete(s.byID, id)
	if build.Token != "" {
		delete(s.byToken, build.Token)
	}
	return true, nil
}

func (s *fakeBuildStore) DeleteExpiredTemp(ctx context.Context, cutoff time.Time) (int64, error) {
	var deleted int64
	for id, build := range s.byID {
		if build.Status == models.BuildStatusTemp && build.ExpiresAt != nil && !build.ExpiresAt.After(cutoff) {
			delete(s.byID, id)
			if build.Token != "" {
				delete(s.byToken, build.Token)
			}
			deleted++
		}
	}
	return deleted, nil
}

func convertParts(parts []models.BuildPartInput) []models.BuildPart {
	result := make([]models.BuildPart, 0, len(parts))
	for _, part := range parts {
		result = append(result, models.BuildPart{
			GearType:      part.GearType,
			CatalogItemID: part.CatalogItemID,
			Position:      part.Position,
			Notes:         part.Notes,
		})
	}
	return result
}

func cloneBuild(build *models.Build) *models.Build {
	if build == nil {
		return nil
	}
	copyBuild := *build
	if build.ExpiresAt != nil {
		expiresAt := *build.ExpiresAt
		copyBuild.ExpiresAt = &expiresAt
	}
	if build.PublishedAt != nil {
		publishedAt := *build.PublishedAt
		copyBuild.PublishedAt = &publishedAt
	}
	if len(build.Parts) > 0 {
		copyBuild.Parts = make([]models.BuildPart, len(build.Parts))
		copy(copyBuild.Parts, build.Parts)
	}
	return &copyBuild
}

func strconvItoa(i int) string {
	if i == 0 {
		return "0"
	}
	var out []byte
	for i > 0 {
		out = append([]byte{byte('0' + i%10)}, out...)
		i /= 10
	}
	return string(out)
}
