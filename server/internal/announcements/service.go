package announcements

import (
	"context"
	"strings"
	"time"

	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

const (
	defaultAdminListLimit = 20
	maxAdminListLimit     = 100
)

type Store interface {
	ListActive(ctx context.Context, placement models.AnnouncementPlacement, audiences []models.AnnouncementAudience, now time.Time) ([]models.Announcement, error)
	ListAdmin(ctx context.Context, params models.AdminAnnouncementListParams) (*models.AnnouncementListResponse, error)
	GetByID(ctx context.Context, id string) (*models.Announcement, error)
	Create(ctx context.Context, actorUserID string, params models.SaveAnnouncementParams) (*models.Announcement, error)
	Update(ctx context.Context, id string, actorUserID string, params models.SaveAnnouncementParams) (*models.Announcement, error)
	Delete(ctx context.Context, id string) (bool, error)
}

type ServiceError struct {
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}

type Service struct {
	store  Store
	logger *logging.Logger
}

func NewService(store Store, logger *logging.Logger) *Service {
	return &Service{
		store:  store,
		logger: logger,
	}
}

func (s *Service) ListActive(ctx context.Context, placement models.AnnouncementPlacement, isAuthenticated bool) ([]models.Announcement, error) {
	if s.store == nil {
		return []models.Announcement{}, nil
	}

	placement = models.NormalizeAnnouncementPlacement(placement)
	if !models.IsValidAnnouncementPlacement(placement) {
		return nil, &ServiceError{Message: "invalid announcement placement"}
	}

	audiences := []models.AnnouncementAudience{models.AnnouncementAudienceAll}
	if isAuthenticated {
		audiences = append(audiences, models.AnnouncementAudienceSignedIn)
	} else {
		audiences = append(audiences, models.AnnouncementAudienceSignedOut)
	}

	items, err := s.store.ListActive(ctx, placement, audiences, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if items == nil {
		return []models.Announcement{}, nil
	}
	return items, nil
}

func (s *Service) ListAdmin(ctx context.Context, params models.AdminAnnouncementListParams) (*models.AnnouncementListResponse, error) {
	if s.store == nil {
		return &models.AnnouncementListResponse{Announcements: []models.Announcement{}}, nil
	}

	params.Query = strings.TrimSpace(params.Query)
	if params.Status != "" {
		params.Status = models.NormalizeAnnouncementStatus(params.Status)
		if !models.IsValidAnnouncementStatus(params.Status) {
			return nil, &ServiceError{Message: "invalid announcement status"}
		}
	}
	params.Limit = normalizeAdminLimit(params.Limit)
	params.Offset = normalizeOffset(params.Offset)

	resp, err := s.store.ListAdmin(ctx, params)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return &models.AnnouncementListResponse{Announcements: []models.Announcement{}}, nil
	}
	if resp.Announcements == nil {
		resp.Announcements = []models.Announcement{}
	}
	return resp, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*models.Announcement, error) {
	if s.store == nil {
		return nil, nil
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, &ServiceError{Message: "announcement id is required"}
	}
	return s.store.GetByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, actorUserID string, params models.SaveAnnouncementParams) (*models.Announcement, error) {
	if s.store == nil {
		return nil, &ServiceError{Message: "announcement store unavailable"}
	}

	normalized, err := normalizeSaveParams(params)
	if err != nil {
		return nil, err
	}

	return s.store.Create(ctx, strings.TrimSpace(actorUserID), normalized)
}

func (s *Service) Update(ctx context.Context, id string, actorUserID string, params models.SaveAnnouncementParams) (*models.Announcement, error) {
	if s.store == nil {
		return nil, &ServiceError{Message: "announcement store unavailable"}
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, &ServiceError{Message: "announcement id is required"}
	}

	normalized, err := normalizeSaveParams(params)
	if err != nil {
		return nil, err
	}

	return s.store.Update(ctx, id, strings.TrimSpace(actorUserID), normalized)
}

func (s *Service) Delete(ctx context.Context, id string) (bool, error) {
	if s.store == nil {
		return false, &ServiceError{Message: "announcement store unavailable"}
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return false, &ServiceError{Message: "announcement id is required"}
	}

	return s.store.Delete(ctx, id)
}

func normalizeSaveParams(params models.SaveAnnouncementParams) (models.SaveAnnouncementParams, error) {
	params.Title = strings.TrimSpace(params.Title)
	params.Body = strings.TrimSpace(params.Body)
	params.Status = models.NormalizeAnnouncementStatus(params.Status)
	params.Audience = models.NormalizeAnnouncementAudience(params.Audience)
	params.CTALabel = strings.TrimSpace(params.CTALabel)
	params.CTAURL = strings.TrimSpace(params.CTAURL)
	params.Placements = normalizePlacements(params.Placements)

	if params.Title == "" {
		return params, &ServiceError{Message: "title is required"}
	}
	if params.Body == "" {
		return params, &ServiceError{Message: "body is required"}
	}
	if !models.IsValidAnnouncementStatus(params.Status) {
		return params, &ServiceError{Message: "invalid announcement status"}
	}
	if !models.IsValidAnnouncementAudience(params.Audience) {
		return params, &ServiceError{Message: "invalid announcement audience"}
	}
	if len(params.Placements) == 0 {
		return params, &ServiceError{Message: "at least one placement is required"}
	}
	if params.CTALabel == "" && params.CTAURL != "" {
		return params, &ServiceError{Message: "cta label is required when cta url is set"}
	}
	if params.CTALabel != "" && params.CTAURL == "" {
		return params, &ServiceError{Message: "cta url is required when cta label is set"}
	}
	if !models.IsValidAnnouncementCTAURL(params.CTAURL) {
		return params, &ServiceError{Message: "cta url must be a relative path or http/https URL"}
	}
	if params.StartsAt != nil {
		start := params.StartsAt.UTC()
		params.StartsAt = &start
	}
	if params.EndsAt != nil {
		end := params.EndsAt.UTC()
		params.EndsAt = &end
	}
	if params.StartsAt != nil && params.EndsAt != nil && params.EndsAt.Before(*params.StartsAt) {
		return params, &ServiceError{Message: "end time must be after start time"}
	}

	return params, nil
}

func normalizePlacements(raw []models.AnnouncementPlacement) []models.AnnouncementPlacement {
	if len(raw) == 0 {
		return nil
	}

	result := make([]models.AnnouncementPlacement, 0, len(raw))
	seen := make(map[models.AnnouncementPlacement]struct{}, len(raw))
	for _, placement := range raw {
		normalized := models.NormalizeAnnouncementPlacement(placement)
		if !models.IsValidAnnouncementPlacement(normalized) {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func normalizeAdminLimit(limit int) int {
	if limit <= 0 {
		return defaultAdminListLimit
	}
	if limit > maxAdminListLimit {
		return maxAdminListLimit
	}
	return limit
}

func normalizeOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}
