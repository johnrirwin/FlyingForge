package models

import (
	"net/url"
	"strings"
	"time"
)

type AnnouncementStatus string

const (
	AnnouncementStatusDraft     AnnouncementStatus = "draft"
	AnnouncementStatusPublished AnnouncementStatus = "published"
	AnnouncementStatusArchived  AnnouncementStatus = "archived"
)

func NormalizeAnnouncementStatus(status AnnouncementStatus) AnnouncementStatus {
	switch strings.ToLower(strings.TrimSpace(string(status))) {
	case string(AnnouncementStatusDraft):
		return AnnouncementStatusDraft
	case string(AnnouncementStatusPublished):
		return AnnouncementStatusPublished
	case string(AnnouncementStatusArchived):
		return AnnouncementStatusArchived
	default:
		return status
	}
}

func IsValidAnnouncementStatus(status AnnouncementStatus) bool {
	switch NormalizeAnnouncementStatus(status) {
	case AnnouncementStatusDraft, AnnouncementStatusPublished, AnnouncementStatusArchived:
		return true
	default:
		return false
	}
}

type AnnouncementPlacement string

const (
	AnnouncementPlacementGlobal    AnnouncementPlacement = "global"
	AnnouncementPlacementHome      AnnouncementPlacement = "home"
	AnnouncementPlacementDashboard AnnouncementPlacement = "dashboard"
	AnnouncementPlacementNews      AnnouncementPlacement = "news"
)

func NormalizeAnnouncementPlacement(placement AnnouncementPlacement) AnnouncementPlacement {
	switch strings.ToLower(strings.TrimSpace(string(placement))) {
	case string(AnnouncementPlacementGlobal):
		return AnnouncementPlacementGlobal
	case string(AnnouncementPlacementHome):
		return AnnouncementPlacementHome
	case string(AnnouncementPlacementDashboard):
		return AnnouncementPlacementDashboard
	case string(AnnouncementPlacementNews):
		return AnnouncementPlacementNews
	default:
		return placement
	}
}

func IsValidAnnouncementPlacement(placement AnnouncementPlacement) bool {
	switch NormalizeAnnouncementPlacement(placement) {
	case AnnouncementPlacementGlobal, AnnouncementPlacementHome, AnnouncementPlacementDashboard, AnnouncementPlacementNews:
		return true
	default:
		return false
	}
}

type AnnouncementAudience string

const (
	AnnouncementAudienceAll       AnnouncementAudience = "all"
	AnnouncementAudienceSignedIn  AnnouncementAudience = "signed_in"
	AnnouncementAudienceSignedOut AnnouncementAudience = "signed_out"
)

func NormalizeAnnouncementAudience(audience AnnouncementAudience) AnnouncementAudience {
	switch strings.ToLower(strings.TrimSpace(string(audience))) {
	case string(AnnouncementAudienceAll):
		return AnnouncementAudienceAll
	case string(AnnouncementAudienceSignedIn):
		return AnnouncementAudienceSignedIn
	case string(AnnouncementAudienceSignedOut):
		return AnnouncementAudienceSignedOut
	default:
		return audience
	}
}

func IsValidAnnouncementAudience(audience AnnouncementAudience) bool {
	switch NormalizeAnnouncementAudience(audience) {
	case AnnouncementAudienceAll, AnnouncementAudienceSignedIn, AnnouncementAudienceSignedOut:
		return true
	default:
		return false
	}
}

type Announcement struct {
	ID          string                  `json:"id"`
	Title       string                  `json:"title"`
	Body        string                  `json:"body"`
	Status      AnnouncementStatus      `json:"status"`
	Priority    int                     `json:"priority"`
	Placements  []AnnouncementPlacement `json:"placements"`
	Audience    AnnouncementAudience    `json:"audience"`
	CTALabel    string                  `json:"ctaLabel,omitempty"`
	CTAURL      string                  `json:"ctaUrl,omitempty"`
	Dismissible bool                    `json:"dismissible"`
	StartsAt    *time.Time              `json:"startsAt,omitempty"`
	EndsAt      *time.Time              `json:"endsAt,omitempty"`
	CreatedAt   time.Time               `json:"createdAt"`
	UpdatedAt   time.Time               `json:"updatedAt"`
}

type SaveAnnouncementParams struct {
	Title       string                  `json:"title"`
	Body        string                  `json:"body"`
	Status      AnnouncementStatus      `json:"status"`
	Priority    int                     `json:"priority"`
	Placements  []AnnouncementPlacement `json:"placements"`
	Audience    AnnouncementAudience    `json:"audience"`
	CTALabel    string                  `json:"ctaLabel,omitempty"`
	CTAURL      string                  `json:"ctaUrl,omitempty"`
	Dismissible bool                    `json:"dismissible"`
	StartsAt    *time.Time              `json:"startsAt,omitempty"`
	EndsAt      *time.Time              `json:"endsAt,omitempty"`
}

type AdminAnnouncementListParams struct {
	Query  string             `json:"query,omitempty"`
	Status AnnouncementStatus `json:"status,omitempty"`
	Limit  int                `json:"limit,omitempty"`
	Offset int                `json:"offset,omitempty"`
}

type AnnouncementListResponse struct {
	Announcements []Announcement `json:"announcements"`
	TotalCount    int            `json:"totalCount"`
}

func IsValidAnnouncementCTAURL(raw string) bool {
	value := strings.TrimSpace(raw)
	if value == "" {
		return true
	}
	if strings.HasPrefix(value, "/") {
		if strings.HasPrefix(value, "//") {
			return false
		}

		parsed, err := url.Parse(value)
		if err != nil {
			return false
		}

		return parsed.Scheme == "" && parsed.Host == ""
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return false
	}

	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return true
	default:
		return false
	}
}
