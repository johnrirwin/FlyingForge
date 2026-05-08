package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/johnrirwin/flyingforge/internal/announcements"
	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

type announcementStoreStub struct {
	listActivePlacement models.AnnouncementPlacement
	listActiveAudiences []models.AnnouncementAudience
	listActiveItems     []models.Announcement

	listAdminParams models.AdminAnnouncementListParams
	listAdminResp   *models.AnnouncementListResponse

	getByIDItem *models.Announcement

	createActor  string
	createParams models.SaveAnnouncementParams
	createItem   *models.Announcement

	updateID     string
	updateActor  string
	updateParams models.SaveAnnouncementParams
	updateItem   *models.Announcement

	deleteID      string
	deleteDeleted bool
}

func (s *announcementStoreStub) ListActive(_ context.Context, placement models.AnnouncementPlacement, audiences []models.AnnouncementAudience, _ time.Time) ([]models.Announcement, error) {
	s.listActivePlacement = placement
	s.listActiveAudiences = append([]models.AnnouncementAudience{}, audiences...)
	return s.listActiveItems, nil
}

func (s *announcementStoreStub) ListAdmin(_ context.Context, params models.AdminAnnouncementListParams) (*models.AnnouncementListResponse, error) {
	s.listAdminParams = params
	if s.listAdminResp != nil {
		return s.listAdminResp, nil
	}
	return &models.AnnouncementListResponse{Announcements: []models.Announcement{}}, nil
}

func (s *announcementStoreStub) GetByID(_ context.Context, id string) (*models.Announcement, error) {
	return s.getByIDItem, nil
}

func (s *announcementStoreStub) Create(_ context.Context, actorUserID string, params models.SaveAnnouncementParams) (*models.Announcement, error) {
	s.createActor = actorUserID
	s.createParams = params
	return s.createItem, nil
}

func (s *announcementStoreStub) Update(_ context.Context, id string, actorUserID string, params models.SaveAnnouncementParams) (*models.Announcement, error) {
	s.updateID = id
	s.updateActor = actorUserID
	s.updateParams = params
	return s.updateItem, nil
}

func (s *announcementStoreStub) Delete(_ context.Context, id string) (bool, error) {
	s.deleteID = id
	return s.deleteDeleted, nil
}

func TestAnnouncementAPIHandleActiveUsesAuthenticatedAudience(t *testing.T) {
	store := &announcementStoreStub{
		listActiveItems: []models.Announcement{{ID: "announcement-1", Title: "Launch"}},
	}
	service := announcements.NewService(store, logging.New(logging.LevelError))
	api := NewAnnouncementAPI(service, nil, logging.New(logging.LevelError))

	req := httptest.NewRequest(http.MethodGet, "/api/announcements/active?placement=news", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "user-1"))
	w := httptest.NewRecorder()

	api.handleActiveAnnouncements(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if store.listActivePlacement != models.AnnouncementPlacementNews {
		t.Fatalf("placement = %q, want %q", store.listActivePlacement, models.AnnouncementPlacementNews)
	}
	if len(store.listActiveAudiences) != 2 || store.listActiveAudiences[1] != models.AnnouncementAudienceSignedIn {
		t.Fatalf("audiences = %v, want signed_in audience", store.listActiveAudiences)
	}

	var response models.AnnouncementListResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.TotalCount != 1 || len(response.Announcements) != 1 {
		t.Fatalf("response = %+v", response)
	}
}

func TestAnnouncementAPIHandleActiveRejectsInvalidPlacement(t *testing.T) {
	store := &announcementStoreStub{}
	service := announcements.NewService(store, logging.New(logging.LevelError))
	api := NewAnnouncementAPI(service, nil, logging.New(logging.LevelError))

	req := httptest.NewRequest(http.MethodGet, "/api/announcements/active?placement=sidebar", nil)
	w := httptest.NewRecorder()

	api.handleActiveAnnouncements(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAdminAPIHandleAdminAnnouncementsCreate(t *testing.T) {
	store := &announcementStoreStub{
		createItem: &models.Announcement{ID: "announcement-1", Title: "Launch"},
	}
	service := announcements.NewService(store, logging.New(logging.LevelError))
	api := &AdminAPI{
		announcementSvc: service,
		logger:          logging.New(logging.LevelError),
	}

	body := bytes.NewBufferString(`{"title":"Launch","body":"Body","status":"published","priority":10,"placements":["home","news"],"audience":"all","ctaLabel":"Read more","ctaUrl":"/news","dismissible":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/announcements", body)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "admin-1"))
	w := httptest.NewRecorder()

	api.handleAdminAnnouncements(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}
	if store.createActor != "admin-1" {
		t.Fatalf("actor = %q, want admin-1", store.createActor)
	}
	if store.createParams.Status != models.AnnouncementStatusPublished {
		t.Fatalf("status = %q, want published", store.createParams.Status)
	}
	if len(store.createParams.Placements) != 2 {
		t.Fatalf("placements = %v", store.createParams.Placements)
	}
}

func TestAdminAPIHandleAdminAnnouncementByIDUpdateAndDelete(t *testing.T) {
	store := &announcementStoreStub{
		updateItem:    &models.Announcement{ID: "announcement-1", Title: "Updated"},
		deleteDeleted: true,
	}
	service := announcements.NewService(store, logging.New(logging.LevelError))
	api := &AdminAPI{
		announcementSvc: service,
		logger:          logging.New(logging.LevelError),
	}

	updateBody := bytes.NewBufferString(`{"title":"Updated","body":"Updated body","status":"published","priority":25,"placements":["dashboard"],"audience":"signed_in","dismissible":false}`)
	updateReq := httptest.NewRequest(http.MethodPut, "/api/admin/announcements/announcement-1", updateBody)
	updateReq = updateReq.WithContext(context.WithValue(updateReq.Context(), auth.UserIDKey, "admin-2"))
	updateW := httptest.NewRecorder()

	api.handleAdminAnnouncementByID(updateW, updateReq)

	if updateW.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d", updateW.Code, http.StatusOK)
	}
	if store.updateID != "announcement-1" {
		t.Fatalf("update id = %q", store.updateID)
	}
	if store.updateActor != "admin-2" {
		t.Fatalf("update actor = %q", store.updateActor)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/admin/announcements/announcement-1", nil)
	deleteW := httptest.NewRecorder()

	api.handleAdminAnnouncementByID(deleteW, deleteReq)

	if deleteW.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want %d", deleteW.Code, http.StatusOK)
	}
	if store.deleteID != "announcement-1" {
		t.Fatalf("delete id = %q", store.deleteID)
	}
}
