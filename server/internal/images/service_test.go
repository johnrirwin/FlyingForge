package images

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/johnrirwin/flyingforge/internal/models"
)

type fakeModerator struct {
	decision *models.ModerationDecision
	err      error
}

func (f *fakeModerator) ModerateImageBytes(ctx context.Context, imageBytes []byte) (*models.ModerationDecision, error) {
	_ = ctx
	_ = imageBytes
	if f.err != nil {
		return nil, f.err
	}
	return f.decision, nil
}

type fakeStorage struct {
	saved []*models.ImageAsset
}

type failingPendingStore struct{}

func (f *failingPendingStore) Put(upload PendingUpload) string {
	_ = upload
	return ""
}

func (f *failingPendingStore) Get(ownerUserID, uploadID string) (*PendingUpload, bool) {
	_ = ownerUserID
	_ = uploadID
	return nil, false
}

func (f *failingPendingStore) Delete(uploadID string) {
	_ = uploadID
}

func (f *fakeStorage) Save(ctx context.Context, req SaveRequest) (*models.ImageAsset, error) {
	_ = ctx
	asset := &models.ImageAsset{
		ID:          "asset-1",
		OwnerUserID: req.OwnerUserID,
		EntityType:  req.EntityType,
		EntityID:    req.EntityID,
		ImageBytes:  req.ImageBytes,
		Status:      models.ImageModerationApproved,
	}
	f.saved = append(f.saved, asset)
	return asset, nil
}

func (f *fakeStorage) Load(ctx context.Context, imageID string) (*models.ImageAsset, error) {
	_ = ctx
	for _, item := range f.saved {
		if item.ID == imageID {
			return item, nil
		}
	}
	return nil, nil
}

func (f *fakeStorage) Delete(ctx context.Context, imageID string) error {
	_ = ctx
	_ = imageID
	return nil
}

func TestServiceModerateUpload(t *testing.T) {
	svc := NewService(
		&fakeModerator{
			decision: &models.ModerationDecision{
				Status: models.ImageModerationApproved,
				Reason: "Approved",
			},
		},
		&fakeStorage{},
		NewInMemoryPendingStore(5*time.Minute),
		5*time.Second,
	)

	decision, uploadID, err := svc.ModerateUpload(context.Background(), "user-1", models.ImageEntityAvatar, []byte("abc"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Status != models.ImageModerationApproved {
		t.Fatalf("status=%s", decision.Status)
	}
	if uploadID == "" {
		t.Fatalf("expected upload id")
	}
}

func TestServiceModerateUploadTimeoutFallback(t *testing.T) {
	svc := NewService(
		&fakeModerator{err: errors.New("boom")},
		&fakeStorage{},
		NewInMemoryPendingStore(5*time.Minute),
		5*time.Second,
	)

	decision, uploadID, err := svc.ModerateUpload(context.Background(), "user-1", models.ImageEntityAvatar, []byte("abc"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Status != models.ImageModerationPendingReview {
		t.Fatalf("status=%s", decision.Status)
	}
	if uploadID != "" {
		t.Fatalf("expected empty upload id")
	}
}

func TestServicePersistApprovedUpload(t *testing.T) {
	store := &fakeStorage{}
	svc := NewService(
		&fakeModerator{
			decision: &models.ModerationDecision{
				Status: models.ImageModerationApproved,
			},
		},
		store,
		NewInMemoryPendingStore(5*time.Minute),
		5*time.Second,
	)

	_, uploadID, err := svc.ModerateUpload(context.Background(), "user-1", models.ImageEntityAvatar, []byte("abc"))
	if err != nil {
		t.Fatalf("moderate upload error: %v", err)
	}
	if uploadID == "" {
		t.Fatalf("expected upload id")
	}

	asset, err := svc.PersistApprovedUpload(context.Background(), "user-1", uploadID, models.ImageEntityAvatar, "entity-1")
	if err != nil {
		t.Fatalf("persist approved upload error: %v", err)
	}
	if asset == nil {
		t.Fatalf("expected saved asset")
	}
	if len(store.saved) != 1 {
		t.Fatalf("expected one save, got %d", len(store.saved))
	}
}

func TestServiceModerateUploadPendingStoreFailure(t *testing.T) {
	svc := NewService(
		&fakeModerator{
			decision: &models.ModerationDecision{
				Status: models.ImageModerationApproved,
				Reason: "Approved",
			},
		},
		&fakeStorage{},
		&failingPendingStore{},
		5*time.Second,
	)

	decision, uploadID, err := svc.ModerateUpload(context.Background(), "user-1", models.ImageEntityAvatar, []byte("abc"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Status != models.ImageModerationPendingReview {
		t.Fatalf("status=%s", decision.Status)
	}
	if uploadID != "" {
		t.Fatalf("expected empty upload id")
	}
}
