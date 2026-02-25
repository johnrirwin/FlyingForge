package builds

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/johnrirwin/flyingforge/internal/images"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

func TestValidateForPublish_MissingRequiredCategories(t *testing.T) {
	build := &models.Build{
		ImageAssetID: "asset-1",
		Description:  "Test build",
		Status:       models.BuildStatusDraft,
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
			name: "valid with stack",
			parts: []models.BuildPart{
				{GearType: models.GearTypeFrame, CatalogItemID: "frame-1", CatalogItem: publishedCatalog("frame-1", models.GearTypeFrame)},
				{GearType: models.GearTypeMotor, CatalogItemID: "motor-1", CatalogItem: publishedCatalog("motor-1", models.GearTypeMotor)},
				{GearType: models.GearTypeStack, CatalogItemID: "stack-1", CatalogItem: publishedCatalog("stack-1", models.GearTypeStack)},
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
			result := ValidateForPublish(&models.Build{
				ImageAssetID: "asset-1",
				Description:  "Test build",
				Parts:        tt.parts,
			})
			if result.Valid != tt.wantValid {
				t.Fatalf("valid=%v want %v, errors=%v", result.Valid, tt.wantValid, result.Errors)
			}
		})
	}
}

func TestIsBuildVerified_RequiresCompletePowerStack(t *testing.T) {
	baseCore := []models.BuildPart{
		{GearType: models.GearTypeFrame, CatalogItemID: "frame-1", CatalogItem: publishedCatalog("frame-1", models.GearTypeFrame)},
		{GearType: models.GearTypeMotor, CatalogItemID: "motor-1", CatalogItem: publishedCatalog("motor-1", models.GearTypeMotor)},
		{GearType: models.GearTypeReceiver, CatalogItemID: "rx-1", CatalogItem: publishedCatalog("rx-1", models.GearTypeReceiver)},
		{GearType: models.GearTypeVTX, CatalogItemID: "vtx-1", CatalogItem: publishedCatalog("vtx-1", models.GearTypeVTX)},
	}

	tests := []struct {
		name     string
		parts    []models.BuildPart
		expected bool
	}{
		{
			name: "verified with aio",
			parts: append(append([]models.BuildPart{}, baseCore...),
				models.BuildPart{GearType: models.GearTypeAIO, CatalogItemID: "aio-1", CatalogItem: publishedCatalog("aio-1", models.GearTypeAIO)},
			),
			expected: true,
		},
		{
			name: "verified with stack",
			parts: append(append([]models.BuildPart{}, baseCore...),
				models.BuildPart{GearType: models.GearTypeStack, CatalogItemID: "stack-1", CatalogItem: publishedCatalog("stack-1", models.GearTypeStack)},
			),
			expected: true,
		},
		{
			name: "verified with fc and esc",
			parts: append(append([]models.BuildPart{}, baseCore...),
				models.BuildPart{GearType: models.GearTypeFC, CatalogItemID: "fc-1", CatalogItem: publishedCatalog("fc-1", models.GearTypeFC)},
				models.BuildPart{GearType: models.GearTypeESC, CatalogItemID: "esc-1", CatalogItem: publishedCatalog("esc-1", models.GearTypeESC)},
			),
			expected: true,
		},
		{
			name: "not verified with fc only",
			parts: append(append([]models.BuildPart{}, baseCore...),
				models.BuildPart{GearType: models.GearTypeFC, CatalogItemID: "fc-1", CatalogItem: publishedCatalog("fc-1", models.GearTypeFC)},
			),
			expected: false,
		},
		{
			name: "not verified with esc only",
			parts: append(append([]models.BuildPart{}, baseCore...),
				models.BuildPart{GearType: models.GearTypeESC, CatalogItemID: "esc-1", CatalogItem: publishedCatalog("esc-1", models.GearTypeESC)},
			),
			expected: false,
		},
		{
			name:     "not verified without power components",
			parts:    append([]models.BuildPart{}, baseCore...),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			build := &models.Build{Parts: tt.parts}
			if got := isBuildVerified(build); got != tt.expected {
				t.Fatalf("isBuildVerified() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestValidateForPublish_FromAircraftRequiresPublishedCatalogParts(t *testing.T) {
	build := &models.Build{
		ImageAssetID:     "asset-1",
		Description:      "Test build",
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

func TestValidateForPublish_FromAircraftRequiresPublishedStackWhenUsed(t *testing.T) {
	build := &models.Build{
		ImageAssetID:     "asset-1",
		Description:      "Test build",
		SourceAircraftID: "aircraft-1",
		Parts: []models.BuildPart{
			{GearType: models.GearTypeFrame, CatalogItemID: "frame-1", CatalogItem: publishedCatalog("frame-1", models.GearTypeFrame)},
			{GearType: models.GearTypeMotor, CatalogItemID: "motor-1", CatalogItem: publishedCatalog("motor-1", models.GearTypeMotor)},
			{GearType: models.GearTypeStack, CatalogItemID: "stack-1", CatalogItem: pendingCatalog("stack-1", models.GearTypeStack)},
			{GearType: models.GearTypeReceiver, CatalogItemID: "rx-1", CatalogItem: publishedCatalog("rx-1", models.GearTypeReceiver)},
			{GearType: models.GearTypeVTX, CatalogItemID: "vtx-1", CatalogItem: publishedCatalog("vtx-1", models.GearTypeVTX)},
		},
	}

	result := ValidateForPublish(build)
	if result.Valid {
		t.Fatalf("expected validation to fail")
	}

	assertHasValidationCode(t, result.Errors, "stack", "not_published")
}

func TestValidateForPublish_FromAircraftChecksAllPresentPowerOptions(t *testing.T) {
	build := &models.Build{
		ImageAssetID:     "asset-1",
		Description:      "Test build",
		SourceAircraftID: "aircraft-1",
		Parts: []models.BuildPart{
			{GearType: models.GearTypeFrame, CatalogItemID: "frame-1", CatalogItem: publishedCatalog("frame-1", models.GearTypeFrame)},
			{GearType: models.GearTypeMotor, CatalogItemID: "motor-1", CatalogItem: publishedCatalog("motor-1", models.GearTypeMotor)},
			{GearType: models.GearTypeAIO, CatalogItemID: "aio-1", CatalogItem: publishedCatalog("aio-1", models.GearTypeAIO)},
			{GearType: models.GearTypeStack, CatalogItemID: "stack-1", CatalogItem: pendingCatalog("stack-1", models.GearTypeStack)},
			{GearType: models.GearTypeFC, CatalogItemID: "fc-1", CatalogItem: publishedCatalog("fc-1", models.GearTypeFC)},
			{GearType: models.GearTypeESC, CatalogItemID: "esc-1", CatalogItem: pendingCatalog("esc-1", models.GearTypeESC)},
			{GearType: models.GearTypeReceiver, CatalogItemID: "rx-1", CatalogItem: publishedCatalog("rx-1", models.GearTypeReceiver)},
			{GearType: models.GearTypeVTX, CatalogItemID: "vtx-1", CatalogItem: publishedCatalog("vtx-1", models.GearTypeVTX)},
		},
	}

	result := ValidateForPublish(build)
	if result.Valid {
		t.Fatalf("expected validation to fail")
	}

	assertHasValidationCode(t, result.Errors, "stack", "not_published")
	assertHasValidationCode(t, result.Errors, "esc", "not_published")
}

func TestValidateForPublish_RequiresDescriptionAndImage(t *testing.T) {
	build := &models.Build{
		Parts: []models.BuildPart{
			{GearType: models.GearTypeFrame, CatalogItemID: "frame-1", CatalogItem: publishedCatalog("frame-1", models.GearTypeFrame)},
			{GearType: models.GearTypeMotor, CatalogItemID: "motor-1", CatalogItem: publishedCatalog("motor-1", models.GearTypeMotor)},
			{GearType: models.GearTypeAIO, CatalogItemID: "aio-1", CatalogItem: publishedCatalog("aio-1", models.GearTypeAIO)},
			{GearType: models.GearTypeReceiver, CatalogItemID: "rx-1", CatalogItem: publishedCatalog("rx-1", models.GearTypeReceiver)},
			{GearType: models.GearTypeVTX, CatalogItemID: "vtx-1", CatalogItem: publishedCatalog("vtx-1", models.GearTypeVTX)},
		},
	}

	result := ValidateForPublish(build)
	if result.Valid {
		t.Fatalf("expected validation to fail")
	}
	assertHasValidationCode(t, result.Errors, "description", "missing_required")
	assertHasValidationCode(t, result.Errors, "image", "missing_required")
}

func TestAircraftComponentCategoryAliasMapping(t *testing.T) {
	tests := []struct {
		name     string
		category models.ComponentCategory
		gearType models.GearType
		eqCat    models.EquipmentCategory
	}{
		{
			name:     "canonical propellers category",
			category: models.ComponentCategoryProps,
			gearType: models.GearTypeProp,
			eqCat:    models.CategoryPropellers,
		},
		{
			name:     "stack component category",
			category: models.ComponentCategoryStack,
			gearType: models.GearTypeStack,
			eqCat:    models.CategoryStacks,
		},
		{
			name:     "plural stacks alias",
			category: models.ComponentCategory("stacks"),
			gearType: models.GearTypeStack,
			eqCat:    models.CategoryStacks,
		},
		{
			name:     "gps component category",
			category: models.ComponentCategoryGPS,
			gearType: models.GearTypeGPS,
			eqCat:    models.CategoryGPS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := aircraftComponentToGearType(tt.category); got != tt.gearType {
				t.Fatalf("aircraftComponentToGearType(%q) = %q, want %q", tt.category, got, tt.gearType)
			}
			if got := componentCategoryToEquipmentCategory(tt.category); got != tt.eqCat {
				t.Fatalf("componentCategoryToEquipmentCategory(%q) = %q, want %q", tt.category, got, tt.eqCat)
			}
		})
	}
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

func TestShareTempByToken_CreatesPermanentSnapshot(t *testing.T) {
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

	shared, err := svc.ShareTempByToken(ctx, created.Token)
	if err != nil {
		t.Fatalf("ShareTempByToken error: %v", err)
	}
	if shared == nil {
		t.Fatalf("expected shared response")
	}
	if shared.Build == nil {
		t.Fatalf("expected shared build in response")
	}
	if shared.Token == "" {
		t.Fatalf("expected shared token")
	}
	if shared.Token == created.Token {
		t.Fatalf("expected new token for shared snapshot")
	}
	if shared.Build.Status != models.BuildStatusShared {
		t.Fatalf("status=%s want SHARED", shared.Build.Status)
	}
	if shared.Build.ExpiresAt != nil {
		t.Fatalf("expected expiresAt to be cleared")
	}
	if !strings.Contains(shared.URL, shared.Token) {
		t.Fatalf("expected URL to contain shared token")
	}

	original, err := svc.GetTempByToken(ctx, created.Token)
	if err != nil {
		t.Fatalf("GetTempByToken error: %v", err)
	}
	if original == nil {
		t.Fatalf("expected original temp build to remain editable")
	}
	if original.Status != models.BuildStatusTemp {
		t.Fatalf("status=%s want TEMP", original.Status)
	}
	if original.ExpiresAt == nil {
		t.Fatalf("expected original temp expiry to remain")
	}

	// Shared snapshots should remain retrievable even when temp cleanup runs.
	deleted, err := svc.CleanupExpiredTemp(ctx)
	if err != nil {
		t.Fatalf("CleanupExpiredTemp error: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("expected cleanup to skip shared build, deleted=%d", deleted)
	}

	fetched, err := svc.GetTempByToken(ctx, shared.Token)
	if err != nil {
		t.Fatalf("GetTempByToken error: %v", err)
	}
	if fetched == nil {
		t.Fatalf("expected shared build to remain retrievable")
	}
	if fetched.Status != models.BuildStatusShared {
		t.Fatalf("status=%s want SHARED", fetched.Status)
	}
}

func TestShareTempByToken_AfterTempUpdateReturnsNewSnapshotURL(t *testing.T) {
	ctx := context.Background()
	store := newFakeBuildStore()
	svc := NewServiceWithDeps(store, nil, nil, logging.New(logging.LevelError))

	created, err := svc.CreateTemp(ctx, "", models.CreateBuildParams{
		Title: "Visitor Build",
		Parts: []models.BuildPartInput{
			{GearType: models.GearTypeFrame, CatalogItemID: "frame-1"},
		},
	})
	if err != nil {
		t.Fatalf("CreateTemp error: %v", err)
	}

	firstShare, err := svc.ShareTempByToken(ctx, created.Token)
	if err != nil {
		t.Fatalf("ShareTempByToken first error: %v", err)
	}
	if firstShare == nil || firstShare.Build == nil {
		t.Fatalf("expected first share response")
	}

	updatedTitle := "Visitor Build v2"
	updatedTemp, err := svc.UpdateTempByToken(ctx, created.Token, models.UpdateBuildParams{
		Title: &updatedTitle,
		Parts: []models.BuildPartInput{
			{GearType: models.GearTypeFrame, CatalogItemID: "frame-1"},
			{GearType: models.GearTypeBattery, CatalogItemID: "battery-1"},
		},
	})
	if err != nil {
		t.Fatalf("UpdateTempByToken error: %v", err)
	}
	if updatedTemp == nil || updatedTemp.Token == "" {
		t.Fatalf("expected rotated temp token")
	}
	if updatedTemp.Token == created.Token {
		t.Fatalf("expected temp token to rotate after update")
	}

	secondShare, err := svc.ShareTempByToken(ctx, updatedTemp.Token)
	if err != nil {
		t.Fatalf("ShareTempByToken second error: %v", err)
	}
	if secondShare == nil || secondShare.Build == nil {
		t.Fatalf("expected second share response")
	}
	if secondShare.Token == firstShare.Token {
		t.Fatalf("expected second share to generate a distinct token")
	}

	firstFetched, err := svc.GetTempByToken(ctx, firstShare.Token)
	if err != nil {
		t.Fatalf("GetTempByToken first share error: %v", err)
	}
	if firstFetched == nil {
		t.Fatalf("expected first shared snapshot to remain")
	}
	if len(firstFetched.Parts) != 1 {
		t.Fatalf("expected first snapshot to retain original parts, got %d", len(firstFetched.Parts))
	}

	oldTemp, err := svc.GetTempByToken(ctx, created.Token)
	if err != nil {
		t.Fatalf("GetTempByToken old temp error: %v", err)
	}
	if oldTemp == nil {
		t.Fatalf("expected old temp token to remain valid until cleanup")
	}
	if len(oldTemp.Parts) != 1 {
		t.Fatalf("expected old temp token to preserve previous parts, got %d", len(oldTemp.Parts))
	}

	secondFetched, err := svc.GetTempByToken(ctx, secondShare.Token)
	if err != nil {
		t.Fatalf("GetTempByToken second share error: %v", err)
	}
	if secondFetched == nil {
		t.Fatalf("expected second shared snapshot to exist")
	}
	if len(secondFetched.Parts) != 2 {
		t.Fatalf("expected second snapshot to include updated parts, got %d", len(secondFetched.Parts))
	}
}

func TestUpdateTempByToken_SharedSnapshotIsReadOnly(t *testing.T) {
	ctx := context.Background()
	store := newFakeBuildStore()
	svc := NewServiceWithDeps(store, nil, nil, logging.New(logging.LevelError))

	created, err := svc.CreateTemp(ctx, "", models.CreateBuildParams{
		Title: "Visitor Build",
		Parts: []models.BuildPartInput{
			{GearType: models.GearTypeFrame, CatalogItemID: "frame-1"},
		},
	})
	if err != nil {
		t.Fatalf("CreateTemp error: %v", err)
	}

	shared, err := svc.ShareTempByToken(ctx, created.Token)
	if err != nil {
		t.Fatalf("ShareTempByToken error: %v", err)
	}
	if shared == nil || shared.Token == "" {
		t.Fatalf("expected shared snapshot token")
	}

	newTitle := "Updated Snapshot Title"
	updated, err := svc.UpdateTempByToken(ctx, shared.Token, models.UpdateBuildParams{Title: &newTitle})
	if err != nil {
		t.Fatalf("UpdateTempByToken error: %v", err)
	}
	if updated != nil {
		t.Fatalf("expected shared snapshot updates to be rejected")
	}

	fetched, err := svc.GetTempByToken(ctx, shared.Token)
	if err != nil {
		t.Fatalf("GetTempByToken error: %v", err)
	}
	if fetched == nil {
		t.Fatalf("expected shared snapshot to still exist")
	}
	if fetched.Title != "Visitor Build" {
		t.Fatalf("expected shared snapshot title to remain unchanged, got %q", fetched.Title)
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

func TestDeleteByOwner_RejectsPublishedBuild(t *testing.T) {
	ctx := context.Background()
	store := newFakeBuildStore()
	svc := NewServiceWithDeps(store, nil, nil, logging.New(logging.LevelError))

	published, err := svc.CreateDraft(ctx, "user-1", models.CreateBuildParams{Title: "Published Build"})
	if err != nil {
		t.Fatalf("CreateDraft error: %v", err)
	}
	if _, err := store.SetStatus(ctx, published.ID, "user-1", models.BuildStatusPublished); err != nil {
		t.Fatalf("SetStatus setup error: %v", err)
	}

	deleted, err := svc.DeleteByOwner(ctx, published.ID, "user-1")
	if err == nil {
		t.Fatalf("expected delete to fail for published build")
	}
	if deleted {
		t.Fatalf("expected published build not to be deleted")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unpublished before deletion") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteByOwner_RejectsPendingReviewBuild(t *testing.T) {
	ctx := context.Background()
	store := newFakeBuildStore()
	svc := NewServiceWithDeps(store, nil, nil, logging.New(logging.LevelError))

	pending, err := svc.CreateDraft(ctx, "user-1", models.CreateBuildParams{Title: "Pending Review Build"})
	if err != nil {
		t.Fatalf("CreateDraft error: %v", err)
	}
	if _, err := store.SetStatus(ctx, pending.ID, "user-1", models.BuildStatusPendingReview); err != nil {
		t.Fatalf("SetStatus setup error: %v", err)
	}

	deleted, err := svc.DeleteByOwner(ctx, pending.ID, "user-1")
	if err == nil {
		t.Fatalf("expected delete to fail for pending review build")
	}
	if deleted {
		t.Fatalf("expected pending review build not to be deleted")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unpublished before deletion") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnpublishForModeration_UnpublishesPublishedBuild(t *testing.T) {
	ctx := context.Background()
	store := newFakeBuildStore()
	svc := NewServiceWithDeps(store, nil, nil, logging.New(logging.LevelError))

	published, err := svc.CreateDraft(ctx, "user-1", models.CreateBuildParams{Title: "Published Build"})
	if err != nil {
		t.Fatalf("CreateDraft error: %v", err)
	}
	if _, err := store.SetStatus(ctx, published.ID, "user-1", models.BuildStatusPublished); err != nil {
		t.Fatalf("SetStatus setup error: %v", err)
	}

	updated, err := svc.UnpublishForModeration(ctx, published.ID)
	if err != nil {
		t.Fatalf("UnpublishForModeration error: %v", err)
	}
	if updated == nil {
		t.Fatalf("expected updated build")
	}
	if updated.Status != models.BuildStatusUnpublished {
		t.Fatalf("status=%s want UNPUBLISHED", updated.Status)
	}
}

func TestUnpublishForModeration_UnpublishesPendingReviewBuild(t *testing.T) {
	ctx := context.Background()
	store := newFakeBuildStore()
	svc := NewServiceWithDeps(store, nil, nil, logging.New(logging.LevelError))

	pending, err := svc.CreateDraft(ctx, "user-1", models.CreateBuildParams{Title: "Pending Review Build"})
	if err != nil {
		t.Fatalf("CreateDraft error: %v", err)
	}
	if _, err := store.SetStatus(ctx, pending.ID, "user-1", models.BuildStatusPendingReview); err != nil {
		t.Fatalf("SetStatus setup error: %v", err)
	}

	updated, err := svc.UnpublishForModeration(ctx, pending.ID)
	if err != nil {
		t.Fatalf("UnpublishForModeration error: %v", err)
	}
	if updated == nil {
		t.Fatalf("expected updated build")
	}
	if updated.Status != models.BuildStatusUnpublished {
		t.Fatalf("status=%s want UNPUBLISHED", updated.Status)
	}
}

func TestDeclineForModeration_RequiresReason(t *testing.T) {
	ctx := context.Background()
	store := newFakeBuildStore()
	svc := NewServiceWithDeps(store, nil, nil, logging.New(logging.LevelError))

	pending, err := svc.CreateDraft(ctx, "user-1", models.CreateBuildParams{Title: "Pending Build"})
	if err != nil {
		t.Fatalf("CreateDraft error: %v", err)
	}
	if _, err := store.SetStatus(ctx, pending.ID, "user-1", models.BuildStatusPendingReview); err != nil {
		t.Fatalf("SetStatus setup error: %v", err)
	}

	_, err = svc.DeclineForModeration(ctx, pending.ID, "   ")
	if err == nil {
		t.Fatalf("expected decline reason validation error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "decline reason is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeclineForModeration_UnpublishesPendingReviewAndStoresReason(t *testing.T) {
	ctx := context.Background()
	store := newFakeBuildStore()
	svc := NewServiceWithDeps(store, nil, nil, logging.New(logging.LevelError))

	pending, err := svc.CreateDraft(ctx, "user-1", models.CreateBuildParams{Title: "Pending Build"})
	if err != nil {
		t.Fatalf("CreateDraft error: %v", err)
	}
	if _, err := store.SetStatus(ctx, pending.ID, "user-1", models.BuildStatusPendingReview); err != nil {
		t.Fatalf("SetStatus setup error: %v", err)
	}

	updated, err := svc.DeclineForModeration(ctx, pending.ID, "Needs more detailed component notes")
	if err != nil {
		t.Fatalf("DeclineForModeration error: %v", err)
	}
	if updated == nil {
		t.Fatalf("expected declined build")
	}
	if updated.Status != models.BuildStatusUnpublished {
		t.Fatalf("status=%s want UNPUBLISHED", updated.Status)
	}
	if updated.ModerationReason != "Needs more detailed component notes" {
		t.Fatalf("moderationReason=%q want %q", updated.ModerationReason, "Needs more detailed component notes")
	}
}

func TestSetReactionAndClearReaction(t *testing.T) {
	ctx := context.Background()
	store := newFakeBuildStore()
	svc := NewServiceWithDeps(store, nil, nil, logging.New(logging.LevelError))

	build, err := svc.CreateDraft(ctx, "owner-1", models.CreateBuildParams{Title: "Reaction Build"})
	if err != nil {
		t.Fatalf("CreateDraft error: %v", err)
	}
	if _, err := store.SetStatus(ctx, build.ID, "owner-1", models.BuildStatusPublished); err != nil {
		t.Fatalf("SetStatus setup error: %v", err)
	}

	liked, err := svc.SetReaction(ctx, build.ID, "viewer-1", models.BuildReactionLike)
	if err != nil {
		t.Fatalf("SetReaction like error: %v", err)
	}
	if liked == nil {
		t.Fatalf("expected updated build")
	}
	if liked.LikeCount != 1 || liked.DislikeCount != 0 {
		t.Fatalf("unexpected reaction counts after like: like=%d dislike=%d", liked.LikeCount, liked.DislikeCount)
	}
	if liked.ViewerReaction != models.BuildReactionLike {
		t.Fatalf("expected viewer reaction LIKE, got %q", liked.ViewerReaction)
	}

	disliked, err := svc.SetReaction(ctx, build.ID, "viewer-1", models.BuildReactionDislike)
	if err != nil {
		t.Fatalf("SetReaction dislike error: %v", err)
	}
	if disliked.LikeCount != 0 || disliked.DislikeCount != 1 {
		t.Fatalf("unexpected reaction counts after dislike: like=%d dislike=%d", disliked.LikeCount, disliked.DislikeCount)
	}
	if disliked.ViewerReaction != models.BuildReactionDislike {
		t.Fatalf("expected viewer reaction DISLIKE, got %q", disliked.ViewerReaction)
	}

	cleared, err := svc.ClearReaction(ctx, build.ID, "viewer-1")
	if err != nil {
		t.Fatalf("ClearReaction error: %v", err)
	}
	if cleared.LikeCount != 0 || cleared.DislikeCount != 0 {
		t.Fatalf("unexpected reaction counts after clear: like=%d dislike=%d", cleared.LikeCount, cleared.DislikeCount)
	}
	if cleared.ViewerReaction != "" {
		t.Fatalf("expected no viewer reaction after clear, got %q", cleared.ViewerReaction)
	}
}

func TestSetReaction_ValidatesInput(t *testing.T) {
	ctx := context.Background()
	store := newFakeBuildStore()
	svc := NewServiceWithDeps(store, nil, nil, logging.New(logging.LevelError))

	if _, err := svc.SetReaction(ctx, "", "viewer-1", models.BuildReactionLike); err == nil || !strings.Contains(err.Error(), "build id is required") {
		t.Fatalf("expected build id validation error, got %v", err)
	}
	if _, err := svc.SetReaction(ctx, "build-1", "", models.BuildReactionLike); err == nil || !strings.Contains(err.Error(), "user id is required") {
		t.Fatalf("expected user id validation error, got %v", err)
	}
	if _, err := svc.SetReaction(ctx, "build-1", "viewer-1", models.BuildReaction("sideways")); err == nil || !strings.Contains(err.Error(), "reaction must be LIKE or DISLIKE") {
		t.Fatalf("expected reaction validation error, got %v", err)
	}
}

func TestPublish_SubmitsPendingReview(t *testing.T) {
	ctx := context.Background()
	store := newFakeBuildStore()
	svc := NewServiceWithDeps(store, nil, nil, logging.New(logging.LevelError))

	created, err := svc.CreateDraft(ctx, "user-1", models.CreateBuildParams{
		Title:       "Moderated Build",
		Description: "Ready for review",
		Parts: []models.BuildPartInput{
			{GearType: models.GearTypeFrame, CatalogItemID: "frame-1"},
			{GearType: models.GearTypeMotor, CatalogItemID: "motor-1"},
			{GearType: models.GearTypeAIO, CatalogItemID: "aio-1"},
			{GearType: models.GearTypeReceiver, CatalogItemID: "rx-1"},
			{GearType: models.GearTypeVTX, CatalogItemID: "vtx-1"},
		},
	})
	if err != nil {
		t.Fatalf("CreateDraft error: %v", err)
	}
	if _, err := store.SetImage(ctx, created.ID, "user-1", "asset-1"); err != nil {
		t.Fatalf("SetImage setup error: %v", err)
	}

	updated, validation, err := svc.Publish(ctx, created.ID, "user-1")
	if err != nil {
		t.Fatalf("Publish error: %v", err)
	}
	if !validation.Valid {
		t.Fatalf("expected validation to pass, errors=%+v", validation.Errors)
	}
	if updated == nil {
		t.Fatalf("expected updated build")
	}
	if updated.Status != models.BuildStatusPendingReview {
		t.Fatalf("status=%s want PENDING_REVIEW", updated.Status)
	}
}

func TestCreateDraft_TrimsYouTubeURL(t *testing.T) {
	store := newFakeBuildStore()
	svc := NewServiceWithDeps(store, nil, nil, logging.New(logging.LevelError))
	ctx := context.Background()

	created, err := svc.CreateDraft(ctx, "user-1", models.CreateBuildParams{
		Title:            "Video Build",
		YouTubeURL:       "  https://www.youtube.com/watch?v=dQw4w9WgXcQ  ",
		FlightYouTubeURL: " https://youtu.be/flight123 ",
	})
	if err != nil {
		t.Fatalf("CreateDraft error: %v", err)
	}
	if created.YouTubeURL != "https://www.youtube.com/watch?v=dQw4w9WgXcQ" {
		t.Fatalf("expected trimmed youtube url, got %q", created.YouTubeURL)
	}
	if created.FlightYouTubeURL != "https://youtu.be/flight123" {
		t.Fatalf("expected trimmed flight youtube url, got %q", created.FlightYouTubeURL)
	}
}

func TestUpdateByOwner_AllowsUpdatingAndClearingYouTubeURL(t *testing.T) {
	store := newFakeBuildStore()
	svc := NewServiceWithDeps(store, nil, nil, logging.New(logging.LevelError))
	ctx := context.Background()

	created, err := svc.CreateDraft(ctx, "user-1", models.CreateBuildParams{
		Title:            "Video Build",
		YouTubeURL:       "https://youtu.be/abc123",
		FlightYouTubeURL: "https://youtu.be/flightabc",
	})
	if err != nil {
		t.Fatalf("CreateDraft error: %v", err)
	}

	nextURL := "  https://www.youtube.com/watch?v=xyz789  "
	updated, err := svc.UpdateByOwner(ctx, created.ID, "user-1", models.UpdateBuildParams{
		YouTubeURL: &nextURL,
	})
	if err != nil {
		t.Fatalf("UpdateByOwner error: %v", err)
	}
	if updated == nil {
		t.Fatalf("expected updated build")
	}
	if updated.YouTubeURL != "https://www.youtube.com/watch?v=xyz789" {
		t.Fatalf("expected trimmed youtube url after update, got %q", updated.YouTubeURL)
	}
	if updated.FlightYouTubeURL != "https://youtu.be/flightabc" {
		t.Fatalf("expected unchanged flight youtube url after update, got %q", updated.FlightYouTubeURL)
	}

	nextFlightURL := "  https://www.youtube.com/watch?v=flightxyz  "
	updatedFlight, err := svc.UpdateByOwner(ctx, created.ID, "user-1", models.UpdateBuildParams{
		FlightYouTubeURL: &nextFlightURL,
	})
	if err != nil {
		t.Fatalf("UpdateByOwner flight error: %v", err)
	}
	if updatedFlight == nil {
		t.Fatalf("expected updated flight build")
	}
	if updatedFlight.FlightYouTubeURL != "https://www.youtube.com/watch?v=flightxyz" {
		t.Fatalf("expected trimmed flight youtube url after update, got %q", updatedFlight.FlightYouTubeURL)
	}

	clearURL := "   "
	cleared, err := svc.UpdateByOwner(ctx, created.ID, "user-1", models.UpdateBuildParams{
		YouTubeURL: &clearURL,
	})
	if err != nil {
		t.Fatalf("UpdateByOwner clear error: %v", err)
	}
	if cleared == nil {
		t.Fatalf("expected cleared build")
	}
	if cleared.YouTubeURL != "" {
		t.Fatalf("expected youtube url to be cleared, got %q", cleared.YouTubeURL)
	}

	clearFlightURL := ""
	clearedFlight, err := svc.UpdateByOwner(ctx, created.ID, "user-1", models.UpdateBuildParams{
		FlightYouTubeURL: &clearFlightURL,
	})
	if err != nil {
		t.Fatalf("UpdateByOwner clear flight error: %v", err)
	}
	if clearedFlight == nil {
		t.Fatalf("expected cleared flight build")
	}
	if clearedFlight.FlightYouTubeURL != "" {
		t.Fatalf("expected flight youtube url to be cleared, got %q", clearedFlight.FlightYouTubeURL)
	}
}

func TestUpdateByOwner_PublishedBuildCreatesDraftRevision(t *testing.T) {
	ctx := context.Background()
	store := newFakeBuildStore()
	svc := NewServiceWithDeps(store, nil, nil, logging.New(logging.LevelError))

	published, err := svc.CreateDraft(ctx, "user-1", models.CreateBuildParams{
		Title:       "Live Build",
		Description: "public description",
		Parts: []models.BuildPartInput{
			{GearType: models.GearTypeFrame, CatalogItemID: "frame-1"},
		},
	})
	if err != nil {
		t.Fatalf("CreateDraft error: %v", err)
	}
	if _, err := store.SetStatus(ctx, published.ID, "user-1", models.BuildStatusPublished); err != nil {
		t.Fatalf("SetStatus setup error: %v", err)
	}

	nextDescription := "draft revision description"
	updated, err := svc.UpdateByOwner(ctx, published.ID, "user-1", models.UpdateBuildParams{
		Description: &nextDescription,
	})
	if err != nil {
		t.Fatalf("UpdateByOwner error: %v", err)
	}
	if updated == nil {
		t.Fatalf("expected updated build")
	}
	if updated.ID == published.ID {
		t.Fatalf("expected a separate revision build id, got same id %q", updated.ID)
	}
	if updated.Status != models.BuildStatusDraft {
		t.Fatalf("expected draft revision status, got %s", updated.Status)
	}
	if updated.Description != nextDescription {
		t.Fatalf("expected revision description %q, got %q", nextDescription, updated.Description)
	}

	stillPublished, err := store.GetForOwner(ctx, published.ID, "user-1")
	if err != nil {
		t.Fatalf("GetForOwner published error: %v", err)
	}
	if stillPublished == nil {
		t.Fatalf("expected published build to remain")
	}
	if stillPublished.Status != models.BuildStatusPublished {
		t.Fatalf("expected original build to remain published, got %s", stillPublished.Status)
	}
	if stillPublished.Description != "public description" {
		t.Fatalf("expected original published description to remain unchanged, got %q", stillPublished.Description)
	}
}

func TestApproveForModeration_MergesPublishedRevision(t *testing.T) {
	ctx := context.Background()
	store := newFakeBuildStore()
	svc := NewServiceWithDeps(store, nil, nil, logging.New(logging.LevelError))

	published, err := svc.CreateDraft(ctx, "user-1", models.CreateBuildParams{
		Title:       "Live Build",
		Description: "public description",
		YouTubeURL:  "https://youtu.be/live",
		Parts: []models.BuildPartInput{
			{GearType: models.GearTypeFrame, CatalogItemID: "frame-1"},
			{GearType: models.GearTypeMotor, CatalogItemID: "motor-1"},
			{GearType: models.GearTypeAIO, CatalogItemID: "aio-1"},
			{GearType: models.GearTypeReceiver, CatalogItemID: "rx-1"},
			{GearType: models.GearTypeVTX, CatalogItemID: "vtx-1"},
		},
	})
	if err != nil {
		t.Fatalf("CreateDraft error: %v", err)
	}
	if _, err := store.SetImage(ctx, published.ID, "user-1", "asset-1"); err != nil {
		t.Fatalf("SetImage setup error: %v", err)
	}
	if _, err := store.SetStatus(ctx, published.ID, "user-1", models.BuildStatusPublished); err != nil {
		t.Fatalf("SetStatus setup error: %v", err)
	}

	nextTitle := "Revised Title"
	nextDescription := "pending description"
	revision, err := svc.UpdateByOwner(ctx, published.ID, "user-1", models.UpdateBuildParams{
		Title:       &nextTitle,
		Description: &nextDescription,
	})
	if err != nil {
		t.Fatalf("UpdateByOwner error: %v", err)
	}
	if revision == nil {
		t.Fatalf("expected revision build")
	}
	if revision.ID == published.ID {
		t.Fatalf("expected separate revision build id")
	}

	beforeApproval, err := store.GetForOwner(ctx, published.ID, "user-1")
	if err != nil {
		t.Fatalf("GetForOwner before approval error: %v", err)
	}
	if beforeApproval == nil || beforeApproval.Title != "Live Build" {
		t.Fatalf("expected published build to remain unchanged before approval, got %+v", beforeApproval)
	}

	if _, validation, err := svc.Publish(ctx, revision.ID, "user-1"); err != nil || !validation.Valid {
		t.Fatalf("Publish revision failed: err=%v validation=%+v", err, validation)
	}

	approved, validation, err := svc.ApproveForModeration(ctx, revision.ID)
	if err != nil {
		t.Fatalf("ApproveForModeration error: %v", err)
	}
	if !validation.Valid {
		t.Fatalf("expected moderation approval validation to pass, got %+v", validation)
	}
	if approved == nil {
		t.Fatalf("expected approved build")
	}
	if approved.ID != published.ID {
		t.Fatalf("expected moderation to update original published build id=%q, got %q", published.ID, approved.ID)
	}
	if approved.Title != nextTitle || approved.Description != nextDescription {
		t.Fatalf("expected approved build to include revision updates, got title=%q description=%q", approved.Title, approved.Description)
	}
	if approved.Status != models.BuildStatusPublished {
		t.Fatalf("expected approved build to remain published, got %s", approved.Status)
	}

	revisionAfterApproval, err := store.GetByID(ctx, revision.ID)
	if err != nil {
		t.Fatalf("GetByID revision after approval error: %v", err)
	}
	if revisionAfterApproval != nil {
		t.Fatalf("expected revision to be removed after approval, got %+v", revisionAfterApproval)
	}
}

func TestApproveForModeration_PublishesPendingBuild(t *testing.T) {
	ctx := context.Background()
	store := newFakeBuildStore()
	svc := NewServiceWithDeps(store, nil, nil, logging.New(logging.LevelError))

	build := &models.Build{
		ID:           "build-1",
		OwnerUserID:  "user-1",
		Status:       models.BuildStatusPendingReview,
		Title:        "Queued Build",
		Description:  "Queued for moderation",
		ImageAssetID: "asset-1",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
		Parts: []models.BuildPart{
			{GearType: models.GearTypeFrame, CatalogItemID: "frame-1", CatalogItem: publishedCatalog("frame-1", models.GearTypeFrame)},
			{GearType: models.GearTypeMotor, CatalogItemID: "motor-1", CatalogItem: publishedCatalog("motor-1", models.GearTypeMotor)},
			{GearType: models.GearTypeAIO, CatalogItemID: "aio-1", CatalogItem: publishedCatalog("aio-1", models.GearTypeAIO)},
			{GearType: models.GearTypeReceiver, CatalogItemID: "rx-1", CatalogItem: publishedCatalog("rx-1", models.GearTypeReceiver)},
			{GearType: models.GearTypeVTX, CatalogItemID: "vtx-1", CatalogItem: publishedCatalog("vtx-1", models.GearTypeVTX)},
		},
	}
	store.byID[build.ID] = cloneBuild(build)

	updated, validation, err := svc.ApproveForModeration(ctx, build.ID)
	if err != nil {
		t.Fatalf("ApproveForModeration error: %v", err)
	}
	if !validation.Valid {
		t.Fatalf("expected validation to pass, errors=%+v", validation.Errors)
	}
	if updated == nil {
		t.Fatalf("expected updated build")
	}
	if updated.Status != models.BuildStatusPublished {
		t.Fatalf("status=%s want PUBLISHED", updated.Status)
	}
	if updated.PublishedAt == nil {
		t.Fatalf("expected publishedAt to be set")
	}
}

func TestApproveForModeration_PublishesUnpublishedBuild(t *testing.T) {
	ctx := context.Background()
	store := newFakeBuildStore()
	svc := NewServiceWithDeps(store, nil, nil, logging.New(logging.LevelError))

	created, err := svc.CreateDraft(ctx, "user-1", models.CreateBuildParams{
		Title:       "Unpublished Build",
		Description: "Ready to republish",
		Parts: []models.BuildPartInput{
			{GearType: models.GearTypeFrame, CatalogItemID: "frame-1"},
			{GearType: models.GearTypeMotor, CatalogItemID: "motor-1"},
			{GearType: models.GearTypeAIO, CatalogItemID: "aio-1"},
			{GearType: models.GearTypeReceiver, CatalogItemID: "rx-1"},
			{GearType: models.GearTypeVTX, CatalogItemID: "vtx-1"},
		},
	})
	if err != nil {
		t.Fatalf("CreateDraft error: %v", err)
	}
	if _, err := store.SetImage(ctx, created.ID, "user-1", "asset-1"); err != nil {
		t.Fatalf("SetImage setup error: %v", err)
	}
	if _, err := store.SetStatus(ctx, created.ID, "user-1", models.BuildStatusUnpublished); err != nil {
		t.Fatalf("SetStatus setup error: %v", err)
	}

	updated, validation, err := svc.ApproveForModeration(ctx, created.ID)
	if err != nil {
		t.Fatalf("ApproveForModeration error: %v", err)
	}
	if !validation.Valid {
		t.Fatalf("expected validation to pass, errors=%+v", validation.Errors)
	}
	if updated == nil {
		t.Fatalf("expected updated build")
	}
	if updated.Status != models.BuildStatusPublished {
		t.Fatalf("status=%s want PUBLISHED", updated.Status)
	}
	if updated.PublishedAt == nil {
		t.Fatalf("expected publishedAt to be set")
	}
}

func TestSetImage_WithApprovedUpload_PersistsAndCleansPreviousAsset(t *testing.T) {
	ctx := context.Background()
	store := newFakeBuildStore()
	svc := NewServiceWithDeps(store, nil, nil, logging.New(logging.LevelError))

	imageSvc := &fakeImagePipeline{
		persistAsset: &models.ImageAsset{
			ID:         "asset-new",
			ImageBytes: []byte{0xFF, 0xD8, 0xFF, 0xDB},
		},
	}
	svc.imageSvc = imageSvc

	build, err := svc.CreateDraft(ctx, "user-1", models.CreateBuildParams{Title: "Image Build"})
	if err != nil {
		t.Fatalf("CreateDraft error: %v", err)
	}
	if _, err := store.SetImage(ctx, build.ID, "user-1", "asset-old"); err != nil {
		t.Fatalf("SetImage setup error: %v", err)
	}

	decision, err := svc.SetImage(ctx, "user-1", models.SetBuildImageParams{
		BuildID:  build.ID,
		UploadID: "upload-1",
	})
	if err != nil {
		t.Fatalf("SetImage error: %v", err)
	}
	if decision == nil || decision.Status != models.ImageModerationApproved {
		t.Fatalf("expected approved decision, got %+v", decision)
	}
	if imageSvc.persistUploadID != "upload-1" || imageSvc.persistEntityType != models.ImageEntityBuild || imageSvc.persistEntityID != build.ID {
		t.Fatalf("unexpected persist args: uploadID=%s entityType=%s entityID=%s", imageSvc.persistUploadID, imageSvc.persistEntityType, imageSvc.persistEntityID)
	}

	updated, err := store.GetForOwner(ctx, build.ID, "user-1")
	if err != nil {
		t.Fatalf("GetForOwner error: %v", err)
	}
	if updated == nil || updated.ImageAssetID != "asset-new" {
		t.Fatalf("expected image asset to be replaced with asset-new, got %+v", updated)
	}
	if len(imageSvc.deletedIDs) != 1 || imageSvc.deletedIDs[0] != "asset-old" {
		t.Fatalf("expected previous asset cleanup, deleted=%v", imageSvc.deletedIDs)
	}
}

func TestSetImage_NonApprovedModeration_DoesNotPersist(t *testing.T) {
	tests := []struct {
		name   string
		status models.ImageModerationStatus
		reason string
	}{
		{name: "rejected", status: models.ImageModerationRejected, reason: "Not allowed"},
		{name: "pending", status: models.ImageModerationPendingReview, reason: "Unable to verify right now"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			store := newFakeBuildStore()
			svc := NewServiceWithDeps(store, nil, nil, logging.New(logging.LevelError))

			imageSvc := &fakeImagePipeline{
				moderateDecision: &models.ModerationDecision{
					Status: tt.status,
					Reason: tt.reason,
				},
			}
			svc.imageSvc = imageSvc

			build, err := svc.CreateDraft(ctx, "user-1", models.CreateBuildParams{Title: "Image Build"})
			if err != nil {
				t.Fatalf("CreateDraft error: %v", err)
			}

			decision, err := svc.SetImage(ctx, "user-1", models.SetBuildImageParams{
				BuildID:   build.ID,
				ImageType: "image/jpeg",
				ImageData: []byte{0xFF, 0xD8, 0xFF, 0xDB},
			})
			if err != nil {
				t.Fatalf("SetImage error: %v", err)
			}
			if decision == nil || decision.Status != tt.status {
				t.Fatalf("expected status=%s got %+v", tt.status, decision)
			}

			updated, err := store.GetForOwner(ctx, build.ID, "user-1")
			if err != nil {
				t.Fatalf("GetForOwner error: %v", err)
			}
			if updated == nil {
				t.Fatalf("expected build")
			}
			if updated.ImageAssetID != "" {
				t.Fatalf("expected no image asset persisted, got %s", updated.ImageAssetID)
			}
		})
	}
}

func TestDeleteImage_RemovesImageAndDeletesAsset(t *testing.T) {
	ctx := context.Background()
	store := newFakeBuildStore()
	svc := NewServiceWithDeps(store, nil, nil, logging.New(logging.LevelError))

	imageSvc := &fakeImagePipeline{}
	svc.imageSvc = imageSvc

	build, err := svc.CreateDraft(ctx, "user-1", models.CreateBuildParams{Title: "Image Build"})
	if err != nil {
		t.Fatalf("CreateDraft error: %v", err)
	}
	if _, err := store.SetImage(ctx, build.ID, "user-1", "asset-old"); err != nil {
		t.Fatalf("SetImage setup error: %v", err)
	}

	if err := svc.DeleteImage(ctx, build.ID, "user-1"); err != nil {
		t.Fatalf("DeleteImage error: %v", err)
	}

	updated, err := store.GetForOwner(ctx, build.ID, "user-1")
	if err != nil {
		t.Fatalf("GetForOwner error: %v", err)
	}
	if updated == nil {
		t.Fatalf("expected build")
	}
	if updated.ImageAssetID != "" {
		t.Fatalf("expected build image to be removed, got %s", updated.ImageAssetID)
	}
	if len(imageSvc.deletedIDs) != 1 || imageSvc.deletedIDs[0] != "asset-old" {
		t.Fatalf("expected old asset to be deleted, deleted=%v", imageSvc.deletedIDs)
	}
}

func TestGetImageAndGetPublicImage_ReturnDetectedContentType(t *testing.T) {
	ctx := context.Background()
	store := newFakeBuildStore()
	svc := NewServiceWithDeps(store, nil, nil, logging.New(logging.LevelError))

	build, err := svc.CreateDraft(ctx, "user-1", models.CreateBuildParams{Title: "Image Build"})
	if err != nil {
		t.Fatalf("CreateDraft error: %v", err)
	}
	if _, err := store.SetImage(ctx, build.ID, "user-1", "asset-image"); err != nil {
		t.Fatalf("SetImage setup error: %v", err)
	}

	imageData, imageType, err := svc.GetImage(ctx, build.ID, "user-1")
	if err != nil {
		t.Fatalf("GetImage error: %v", err)
	}
	if len(imageData) == 0 {
		t.Fatalf("expected image data for owner")
	}
	if imageType == "" {
		t.Fatalf("expected detected content type for owner image")
	}

	if _, err := store.SetStatus(ctx, build.ID, "user-1", models.BuildStatusPublished); err != nil {
		t.Fatalf("SetStatus setup error: %v", err)
	}

	publicData, publicType, err := svc.GetPublicImage(ctx, build.ID)
	if err != nil {
		t.Fatalf("GetPublicImage error: %v", err)
	}
	if len(publicData) == 0 {
		t.Fatalf("expected public image data")
	}
	if publicType == "" {
		t.Fatalf("expected detected content type for public image")
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
	byID                map[string]*models.Build
	byToken             map[string]string
	reactions           map[string]map[string]models.BuildReaction
	revisionByPublished map[string]string
	publishedByRevision map[string]string
	nextID              int
}

func newFakeBuildStore() *fakeBuildStore {
	return &fakeBuildStore{
		byID:                map[string]*models.Build{},
		byToken:             map[string]string{},
		reactions:           map[string]map[string]models.BuildReaction{},
		revisionByPublished: map[string]string{},
		publishedByRevision: map[string]string{},
	}
}

func (s *fakeBuildStore) Create(ctx context.Context, ownerUserID string, status models.BuildStatus, title string, description string, youtubeURL string, flightYouTubeURL string, sourceAircraftID string, token string, expiresAt *time.Time, parts []models.BuildPartInput) (*models.Build, error) {
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
		YouTubeURL:       youtubeURL,
		FlightYouTubeURL: flightYouTubeURL,
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
		if build.OwnerUserID == ownerUserID &&
			(build.Status == models.BuildStatusDraft || build.Status == models.BuildStatusPendingReview || build.Status == models.BuildStatusPublished || build.Status == models.BuildStatusUnpublished) {
			items = append(items, *cloneBuild(build))
		}
	}
	return &models.BuildListResponse{Builds: items, TotalCount: len(items)}, nil
}

func (s *fakeBuildStore) ListPublic(ctx context.Context, params models.BuildListParams, viewerUserID string) (*models.BuildListResponse, error) {
	items := make([]models.Build, 0)
	for _, build := range s.byID {
		if build.Status == models.BuildStatusPublished {
			next := cloneBuild(build)
			s.applyReactionMeta(next, viewerUserID)
			items = append(items, *next)
		}
	}
	return &models.BuildListResponse{Builds: items, TotalCount: len(items)}, nil
}

func (s *fakeBuildStore) ListPublishedByOwner(ctx context.Context, ownerUserID string, viewerUserID string, limit int) ([]models.Build, error) {
	items := make([]models.Build, 0)
	for _, build := range s.byID {
		if build.OwnerUserID == ownerUserID && build.Status == models.BuildStatusPublished {
			next := cloneBuild(build)
			s.applyReactionMeta(next, viewerUserID)
			items = append(items, *next)
		}
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *fakeBuildStore) ListForModeration(ctx context.Context, params models.BuildModerationListParams) (*models.BuildListResponse, error) {
	status := models.NormalizeBuildStatus(params.Status)
	if status == "" {
		status = models.BuildStatusPendingReview
	}
	items := make([]models.Build, 0)
	for _, build := range s.byID {
		if build.Status == status {
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

func (s *fakeBuildStore) GetPublic(ctx context.Context, id string, viewerUserID string) (*models.Build, error) {
	build := s.byID[id]
	if build == nil || build.Status != models.BuildStatusPublished {
		return nil, nil
	}
	next := cloneBuild(build)
	s.applyReactionMeta(next, viewerUserID)
	return next, nil
}

func (s *fakeBuildStore) GetForModeration(ctx context.Context, id string) (*models.Build, error) {
	build := s.byID[id]
	if build == nil {
		return nil, nil
	}
	switch build.Status {
	case models.BuildStatusDraft, models.BuildStatusPendingReview, models.BuildStatusPublished, models.BuildStatusUnpublished:
		return cloneBuild(build), nil
	default:
		return nil, nil
	}
}

func (s *fakeBuildStore) GetTempByToken(ctx context.Context, token string) (*models.Build, error) {
	id, ok := s.byToken[token]
	if !ok {
		return nil, nil
	}
	build := s.byID[id]
	if build == nil {
		return nil, nil
	}
	if build.Status == models.BuildStatusShared {
		return cloneBuild(build), nil
	}
	if build.Status != models.BuildStatusTemp {
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
	target := build
	if build.Status == models.BuildStatusPublished {
		revisionID := s.revisionByPublished[build.ID]
		if revisionID != "" {
			if existing := s.byID[revisionID]; existing != nil && existing.OwnerUserID == ownerUserID {
				target = existing
			}
		}

		if target == build {
			s.nextID++
			revisionID = "build-" + strconvItoa(s.nextID)
			now := time.Now().UTC()
			revision := cloneBuild(build)
			revision.ID = revisionID
			revision.Status = models.BuildStatusDraft
			revision.PublishedAt = nil
			revision.CreatedAt = now
			revision.UpdatedAt = now
			s.byID[revisionID] = revision
			s.revisionByPublished[build.ID] = revisionID
			s.publishedByRevision[revisionID] = build.ID
			target = revision
		}
	}
	if params.Title != nil {
		target.Title = *params.Title
	}
	if params.Description != nil {
		target.Description = *params.Description
	}
	if params.YouTubeURL != nil {
		target.YouTubeURL = *params.YouTubeURL
	}
	if params.FlightYouTubeURL != nil {
		target.FlightYouTubeURL = *params.FlightYouTubeURL
	}
	if params.Parts != nil {
		target.Parts = convertParts(params.Parts)
	}
	target.UpdatedAt = time.Now().UTC()
	return cloneBuild(target), nil
}

func (s *fakeBuildStore) UpdateTempByToken(ctx context.Context, token string, params models.UpdateBuildParams, nextToken string) (*models.Build, error) {
	id, ok := s.byToken[token]
	if !ok {
		return nil, nil
	}
	build := s.byID[id]
	if build == nil || build.Status != models.BuildStatusTemp {
		return nil, nil
	}

	title := build.Title
	if params.Title != nil {
		title = *params.Title
	}
	description := build.Description
	if params.Description != nil {
		description = *params.Description
	}
	youtubeURL := build.YouTubeURL
	if params.YouTubeURL != nil {
		youtubeURL = *params.YouTubeURL
	}
	flightYouTubeURL := build.FlightYouTubeURL
	if params.FlightYouTubeURL != nil {
		flightYouTubeURL = *params.FlightYouTubeURL
	}

	parts := make([]models.BuildPart, len(build.Parts))
	copy(parts, build.Parts)
	if params.Parts != nil {
		parts = convertParts(params.Parts)
	}

	s.nextID++
	newID := "build-" + strconvItoa(s.nextID)
	now := time.Now().UTC()
	var expiresAt *time.Time
	if build.ExpiresAt != nil {
		copyExpiry := *build.ExpiresAt
		expiresAt = &copyExpiry
	}

	next := &models.Build{
		ID:               newID,
		OwnerUserID:      build.OwnerUserID,
		Status:           models.BuildStatusTemp,
		Token:            nextToken,
		ExpiresAt:        expiresAt,
		Title:            title,
		Description:      description,
		YouTubeURL:       youtubeURL,
		FlightYouTubeURL: flightYouTubeURL,
		SourceAircraftID: build.SourceAircraftID,
		CreatedAt:        now,
		UpdatedAt:        now,
		Parts:            parts,
	}

	s.byID[newID] = cloneBuild(next)
	s.byToken[nextToken] = newID
	return cloneBuild(next), nil
}

func (s *fakeBuildStore) UpdateForModeration(ctx context.Context, id string, params models.UpdateBuildParams) (*models.Build, error) {
	build := s.byID[id]
	if build == nil {
		return nil, nil
	}
	if params.Title != nil {
		build.Title = *params.Title
	}
	if params.Description != nil {
		build.Description = *params.Description
	}
	if params.YouTubeURL != nil {
		build.YouTubeURL = *params.YouTubeURL
	}
	if params.FlightYouTubeURL != nil {
		build.FlightYouTubeURL = *params.FlightYouTubeURL
	}
	if params.Parts != nil {
		build.Parts = convertParts(params.Parts)
	}
	build.UpdatedAt = time.Now().UTC()
	return cloneBuild(build), nil
}

func (s *fakeBuildStore) ShareTempByToken(ctx context.Context, token string) (*models.Build, error) {
	id, ok := s.byToken[token]
	if !ok {
		return nil, nil
	}
	build := s.byID[id]
	if build == nil {
		return nil, nil
	}
	if build.Status == models.BuildStatusShared {
		return cloneBuild(build), nil
	}
	if build.Status != models.BuildStatusTemp {
		return nil, nil
	}
	if build.ExpiresAt != nil && build.ExpiresAt.Before(time.Now().UTC()) {
		return nil, nil
	}
	build.Status = models.BuildStatusShared
	build.ExpiresAt = nil
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
	switch status {
	case models.BuildStatusPendingReview:
		build.PublishedAt = nil
		build.ModerationReason = ""
	case models.BuildStatusPublished:
		build.PublishedAt = &now
		build.ModerationReason = ""
	case models.BuildStatusUnpublished:
		build.PublishedAt = nil
		build.ModerationReason = ""
	}
	build.UpdatedAt = now
	return cloneBuild(build), nil
}

func (s *fakeBuildStore) SetReaction(ctx context.Context, id string, userID string, reaction models.BuildReaction) (*models.Build, error) {
	build := s.byID[id]
	if build == nil || build.Status != models.BuildStatusPublished {
		return nil, nil
	}

	reaction = models.NormalizeBuildReaction(reaction)
	if reaction != models.BuildReactionLike && reaction != models.BuildReactionDislike {
		return nil, fmt.Errorf("invalid reaction")
	}

	if _, ok := s.reactions[id]; !ok {
		s.reactions[id] = map[string]models.BuildReaction{}
	}
	s.reactions[id][userID] = reaction
	build.UpdatedAt = time.Now().UTC()

	next := cloneBuild(build)
	s.applyReactionMeta(next, userID)
	return next, nil
}

func (s *fakeBuildStore) ClearReaction(ctx context.Context, id string, userID string) (*models.Build, error) {
	build := s.byID[id]
	if build == nil || build.Status != models.BuildStatusPublished {
		return nil, nil
	}

	if reactionsByUser, ok := s.reactions[id]; ok {
		delete(reactionsByUser, userID)
		if len(reactionsByUser) == 0 {
			delete(s.reactions, id)
		}
	}
	build.UpdatedAt = time.Now().UTC()

	next := cloneBuild(build)
	s.applyReactionMeta(next, userID)
	return next, nil
}

func (s *fakeBuildStore) ApproveForModeration(ctx context.Context, id string) (*models.Build, error) {
	build := s.byID[id]
	if build == nil || build.Status != models.BuildStatusPendingReview {
		return nil, nil
	}

	if publishedID := s.publishedByRevision[id]; publishedID != "" {
		publishedBuild := s.byID[publishedID]
		if publishedBuild == nil || publishedBuild.Status != models.BuildStatusPublished {
			return nil, nil
		}
		now := time.Now().UTC()
		publishedBuild.Title = build.Title
		publishedBuild.Description = build.Description
		publishedBuild.YouTubeURL = build.YouTubeURL
		publishedBuild.FlightYouTubeURL = build.FlightYouTubeURL
		publishedBuild.SourceAircraftID = build.SourceAircraftID
		publishedBuild.ImageAssetID = build.ImageAssetID
		publishedBuild.ModerationReason = ""
		publishedBuild.Parts = convertParts(models.BuildPartInputsFromParts(build.Parts))
		publishedBuild.UpdatedAt = now

		delete(s.byID, id)
		delete(s.publishedByRevision, id)
		delete(s.revisionByPublished, publishedID)

		return cloneBuild(publishedBuild), nil
	}

	now := time.Now().UTC()
	build.Status = models.BuildStatusPublished
	build.PublishedAt = &now
	build.ModerationReason = ""
	build.UpdatedAt = now
	return cloneBuild(build), nil
}

func (s *fakeBuildStore) DeclineForModeration(ctx context.Context, id string, reason string) (*models.Build, error) {
	build := s.byID[id]
	if build == nil || build.Status != models.BuildStatusPendingReview {
		return nil, nil
	}
	build.Status = models.BuildStatusUnpublished
	build.PublishedAt = nil
	build.ModerationReason = strings.TrimSpace(reason)
	build.UpdatedAt = time.Now().UTC()
	return cloneBuild(build), nil
}

func (s *fakeBuildStore) SetImage(ctx context.Context, id string, ownerUserID string, imageAssetID string) (string, error) {
	build := s.byID[id]
	if build == nil || build.OwnerUserID != ownerUserID {
		return "", fmt.Errorf("build not found")
	}
	prev := build.ImageAssetID
	build.ImageAssetID = imageAssetID
	build.UpdatedAt = time.Now().UTC()
	return prev, nil
}

func (s *fakeBuildStore) SetImageForModeration(ctx context.Context, id string, imageAssetID string) (string, error) {
	build := s.byID[id]
	if build == nil {
		return "", fmt.Errorf("build not found")
	}
	prev := build.ImageAssetID
	build.ImageAssetID = imageAssetID
	build.UpdatedAt = time.Now().UTC()
	return prev, nil
}

func (s *fakeBuildStore) GetImageForOwner(ctx context.Context, id string, ownerUserID string) ([]byte, error) {
	build := s.byID[id]
	if build == nil || build.OwnerUserID != ownerUserID || build.ImageAssetID == "" {
		return nil, nil
	}
	return []byte("image"), nil
}

func (s *fakeBuildStore) GetPublicImage(ctx context.Context, id string) ([]byte, error) {
	build := s.byID[id]
	if build == nil || build.Status != models.BuildStatusPublished || build.ImageAssetID == "" {
		return nil, nil
	}
	return []byte("image"), nil
}

func (s *fakeBuildStore) GetImageForModeration(ctx context.Context, id string) ([]byte, error) {
	build := s.byID[id]
	if build == nil || build.ImageAssetID == "" {
		return nil, nil
	}
	return []byte("image"), nil
}

func (s *fakeBuildStore) DeleteImage(ctx context.Context, id string, ownerUserID string) (string, error) {
	build := s.byID[id]
	if build == nil || build.OwnerUserID != ownerUserID {
		return "", fmt.Errorf("build not found")
	}
	prev := build.ImageAssetID
	build.ImageAssetID = ""
	build.UpdatedAt = time.Now().UTC()
	return prev, nil
}

func (s *fakeBuildStore) DeleteImageForModeration(ctx context.Context, id string) (string, error) {
	build := s.byID[id]
	if build == nil {
		return "", fmt.Errorf("build not found")
	}
	prev := build.ImageAssetID
	build.ImageAssetID = ""
	build.UpdatedAt = time.Now().UTC()
	return prev, nil
}

func (s *fakeBuildStore) Delete(ctx context.Context, id string, ownerUserID string) (bool, error) {
	build := s.byID[id]
	if build == nil || build.OwnerUserID != ownerUserID {
		return false, nil
	}
	if build.Status != models.BuildStatusDraft && build.Status != models.BuildStatusPublished && build.Status != models.BuildStatusUnpublished {
		if build.Status != models.BuildStatusPendingReview {
			return false, nil
		}
	}
	delete(s.byID, id)
	if build.Token != "" {
		delete(s.byToken, build.Token)
	}
	if revisionID := s.revisionByPublished[id]; revisionID != "" {
		delete(s.byID, revisionID)
		delete(s.publishedByRevision, revisionID)
		delete(s.revisionByPublished, id)
	}
	if publishedID := s.publishedByRevision[id]; publishedID != "" {
		delete(s.publishedByRevision, id)
		delete(s.revisionByPublished, publishedID)
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

func (s *fakeBuildStore) applyReactionMeta(build *models.Build, viewerUserID string) {
	if build == nil {
		return
	}

	build.LikeCount = 0
	build.DislikeCount = 0
	build.ViewerReaction = ""

	reactionsByUser := s.reactions[build.ID]
	for _, reaction := range reactionsByUser {
		switch reaction {
		case models.BuildReactionLike:
			build.LikeCount++
		case models.BuildReactionDislike:
			build.DislikeCount++
		}
	}

	if viewerUserID != "" {
		build.ViewerReaction = reactionsByUser[viewerUserID]
	}
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

type fakeImagePipeline struct {
	moderateDecision *models.ModerationDecision
	moderateAsset    *models.ImageAsset
	moderateErr      error

	persistAsset      *models.ImageAsset
	persistErr        error
	persistOwnerUser  string
	persistUploadID   string
	persistEntityType models.ImageEntityType
	persistEntityID   string

	deletedIDs []string
}

func (f *fakeImagePipeline) ModerateAndPersist(ctx context.Context, req images.SaveRequest) (*models.ModerationDecision, *models.ImageAsset, error) {
	if f.moderateErr != nil {
		return nil, nil, f.moderateErr
	}
	if f.moderateDecision != nil {
		return f.moderateDecision, f.moderateAsset, nil
	}
	return &models.ModerationDecision{
			Status: models.ImageModerationApproved,
			Reason: "Approved",
		}, &models.ImageAsset{
			ID:         "asset-generated",
			ImageBytes: req.ImageBytes,
		}, nil
}

func (f *fakeImagePipeline) PersistApprovedUpload(ctx context.Context, ownerUserID, uploadID string, entityType models.ImageEntityType, entityID string) (*models.ImageAsset, error) {
	if f.persistErr != nil {
		return nil, f.persistErr
	}
	f.persistOwnerUser = ownerUserID
	f.persistUploadID = uploadID
	f.persistEntityType = entityType
	f.persistEntityID = entityID
	if f.persistAsset != nil {
		return f.persistAsset, nil
	}
	return &models.ImageAsset{
		ID:         "asset-from-upload",
		ImageBytes: []byte{0xFF, 0xD8, 0xFF, 0xDB},
	}, nil
}

func (f *fakeImagePipeline) Delete(ctx context.Context, imageID string) error {
	f.deletedIDs = append(f.deletedIDs, imageID)
	return nil
}
