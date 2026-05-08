package announcements

import (
	"context"
	"testing"
	"time"

	"github.com/johnrirwin/flyingforge/internal/models"
)

type stubStore struct {
	listActivePlacement models.AnnouncementPlacement
	listActiveAudiences []models.AnnouncementAudience
	listActiveNow       time.Time
	listActiveItems     []models.Announcement

	createActor  string
	createParams models.SaveAnnouncementParams
	createItem   *models.Announcement
}

func (s *stubStore) ListActive(_ context.Context, placement models.AnnouncementPlacement, audiences []models.AnnouncementAudience, now time.Time) ([]models.Announcement, error) {
	s.listActivePlacement = placement
	s.listActiveAudiences = append([]models.AnnouncementAudience{}, audiences...)
	s.listActiveNow = now
	return s.listActiveItems, nil
}

func (s *stubStore) ListAdmin(_ context.Context, params models.AdminAnnouncementListParams) (*models.AnnouncementListResponse, error) {
	return &models.AnnouncementListResponse{Announcements: []models.Announcement{}, TotalCount: 0}, nil
}

func (s *stubStore) GetByID(_ context.Context, id string) (*models.Announcement, error) {
	return nil, nil
}

func (s *stubStore) Create(_ context.Context, actorUserID string, params models.SaveAnnouncementParams) (*models.Announcement, error) {
	s.createActor = actorUserID
	s.createParams = params
	return s.createItem, nil
}

func (s *stubStore) Update(_ context.Context, id string, actorUserID string, params models.SaveAnnouncementParams) (*models.Announcement, error) {
	s.createActor = actorUserID
	s.createParams = params
	return s.createItem, nil
}

func (s *stubStore) Delete(_ context.Context, id string) (bool, error) {
	return true, nil
}

func TestServiceListActiveChoosesAudienceBucket(t *testing.T) {
	store := &stubStore{
		listActiveItems: []models.Announcement{{ID: "announcement-1", Title: "Launch"}},
	}
	service := NewService(store, nil)

	_, err := service.ListActive(context.Background(), models.AnnouncementPlacementHome, false)
	if err != nil {
		t.Fatalf("ListActive() error = %v", err)
	}

	if store.listActivePlacement != models.AnnouncementPlacementHome {
		t.Fatalf("placement = %q, want %q", store.listActivePlacement, models.AnnouncementPlacementHome)
	}

	expected := []models.AnnouncementAudience{
		models.AnnouncementAudienceAll,
		models.AnnouncementAudienceSignedOut,
	}
	if len(store.listActiveAudiences) != len(expected) {
		t.Fatalf("audience length = %d, want %d", len(store.listActiveAudiences), len(expected))
	}
	for i := range expected {
		if store.listActiveAudiences[i] != expected[i] {
			t.Fatalf("audience[%d] = %q, want %q", i, store.listActiveAudiences[i], expected[i])
		}
	}

	_, err = service.ListActive(context.Background(), models.AnnouncementPlacementHome, true)
	if err != nil {
		t.Fatalf("ListActive() authenticated error = %v", err)
	}
	if store.listActiveAudiences[1] != models.AnnouncementAudienceSignedIn {
		t.Fatalf("authenticated audience = %q, want %q", store.listActiveAudiences[1], models.AnnouncementAudienceSignedIn)
	}
	if store.listActiveNow.IsZero() {
		t.Fatal("expected service to pass current time to store")
	}
}

func TestServiceCreateNormalizesAndValidates(t *testing.T) {
	start := time.Date(2026, 5, 7, 13, 0, 0, 0, time.FixedZone("EDT", -4*60*60))
	end := start.Add(2 * time.Hour)

	store := &stubStore{
		createItem: &models.Announcement{ID: "announcement-1"},
	}
	service := NewService(store, nil)

	_, err := service.Create(context.Background(), " admin-1 ", models.SaveAnnouncementParams{
		Title:       "  MCP integrations are now available  ",
		Body:        "  You can now connect ChatGPT to FlyingForge via MCP.  ",
		Status:      "PUBLISHED",
		Priority:    50,
		Placements:  []models.AnnouncementPlacement{"news", "GLOBAL", "news"},
		Audience:    "SIGNED_IN",
		CTALabel:    " Learn more ",
		CTAURL:      " /news ",
		Dismissible: true,
		StartsAt:    &start,
		EndsAt:      &end,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if store.createActor != "admin-1" {
		t.Fatalf("actor = %q, want admin-1", store.createActor)
	}
	if store.createParams.Title != "MCP integrations are now available" {
		t.Fatalf("title = %q", store.createParams.Title)
	}
	if store.createParams.Body != "You can now connect ChatGPT to FlyingForge via MCP." {
		t.Fatalf("body = %q", store.createParams.Body)
	}
	if store.createParams.Status != models.AnnouncementStatusPublished {
		t.Fatalf("status = %q", store.createParams.Status)
	}
	if store.createParams.Audience != models.AnnouncementAudienceSignedIn {
		t.Fatalf("audience = %q", store.createParams.Audience)
	}
	if len(store.createParams.Placements) != 2 {
		t.Fatalf("placements = %v, want 2 unique placements", store.createParams.Placements)
	}
	if store.createParams.StartsAt == nil || store.createParams.EndsAt == nil {
		t.Fatal("expected schedule to be preserved")
	}
	if store.createParams.StartsAt.Location() != time.UTC || store.createParams.EndsAt.Location() != time.UTC {
		t.Fatal("expected service to normalize schedule to UTC")
	}
}

func TestServiceCreateRejectsInvalidScheduleAndCTA(t *testing.T) {
	service := NewService(&stubStore{}, nil)
	start := time.Date(2026, 5, 7, 14, 0, 0, 0, time.UTC)
	end := start.Add(-time.Minute)

	_, err := service.Create(context.Background(), "admin-1", models.SaveAnnouncementParams{
		Title:       "Launch",
		Body:        "Body",
		Status:      models.AnnouncementStatusPublished,
		Placements:  []models.AnnouncementPlacement{models.AnnouncementPlacementHome},
		Audience:    models.AnnouncementAudienceAll,
		Dismissible: true,
		CTALabel:    "Learn more",
		CTAURL:      "javascript:alert(1)",
		StartsAt:    &start,
		EndsAt:      &end,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
