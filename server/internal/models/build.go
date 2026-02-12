package models

import (
	"strings"
	"time"
)

// BuildStatus describes lifecycle state for a build.
type BuildStatus string

const (
	BuildStatusTemp          BuildStatus = "TEMP"
	BuildStatusShared        BuildStatus = "SHARED"
	BuildStatusDraft         BuildStatus = "DRAFT"
	BuildStatusPendingReview BuildStatus = "PENDING_REVIEW"
	BuildStatusPublished     BuildStatus = "PUBLISHED"
	BuildStatusUnpublished   BuildStatus = "UNPUBLISHED"
)

// NormalizeBuildStatus canonicalizes user-provided status values.
func NormalizeBuildStatus(status BuildStatus) BuildStatus {
	switch strings.ToUpper(strings.TrimSpace(string(status))) {
	case string(BuildStatusTemp):
		return BuildStatusTemp
	case string(BuildStatusShared):
		return BuildStatusShared
	case string(BuildStatusDraft):
		return BuildStatusDraft
	case string(BuildStatusPendingReview):
		return BuildStatusPendingReview
	case string(BuildStatusPublished):
		return BuildStatusPublished
	case string(BuildStatusUnpublished):
		return BuildStatusUnpublished
	default:
		return status
	}
}

// BuildSort controls public build ordering.
type BuildSort string

const (
	BuildSortNewest BuildSort = "newest"
)

// BuildPartInput is a request payload for setting a build part.
type BuildPartInput struct {
	GearType      GearType `json:"gearType"`
	CatalogItemID string   `json:"catalogItemId"`
	Position      int      `json:"position,omitempty"`
	Notes         string   `json:"notes,omitempty"`
}

// BuildPart stores an assigned part for a build.
type BuildPart struct {
	ID            string            `json:"id,omitempty"`
	BuildID       string            `json:"buildId,omitempty"`
	GearType      GearType          `json:"gearType"`
	CatalogItemID string            `json:"catalogItemId,omitempty"`
	Position      int               `json:"position,omitempty"`
	Notes         string            `json:"notes,omitempty"`
	CreatedAt     time.Time         `json:"createdAt,omitempty"`
	UpdatedAt     time.Time         `json:"updatedAt,omitempty"`
	CatalogItem   *BuildCatalogItem `json:"catalogItem,omitempty"`
}

// BuildCatalogItem is a minimal catalog payload embedded on build parts.
type BuildCatalogItem struct {
	ID       string            `json:"id"`
	GearType GearType          `json:"gearType"`
	Brand    string            `json:"brand"`
	Model    string            `json:"model"`
	Variant  string            `json:"variant,omitempty"`
	Status   CatalogItemStatus `json:"status"`
	ImageURL string            `json:"imageUrl,omitempty"`
}

// DisplayName returns a formatted catalog item name.
func (i *BuildCatalogItem) DisplayName() string {
	if i == nil {
		return ""
	}
	name := strings.TrimSpace(strings.TrimSpace(i.Brand + " " + i.Model))
	if i.Variant != "" {
		name = strings.TrimSpace(name + " " + i.Variant)
	}
	return name
}

// BuildPilot summarizes owner identity for public build views.
type BuildPilot struct {
	UserID          string `json:"userId,omitempty"`
	CallSign        string `json:"callSign,omitempty"`
	DisplayName     string `json:"displayName,omitempty"`
	IsProfilePublic bool   `json:"isProfilePublic"`
	ProfileURL      string `json:"profileUrl,omitempty"`
}

// DisplayName resolves the best available pilot label.
func (p *BuildPilot) DisplayNameOrDefault() string {
	if p == nil {
		return "Pilot"
	}
	if strings.TrimSpace(p.CallSign) != "" {
		return strings.TrimSpace(p.CallSign)
	}
	if strings.TrimSpace(p.DisplayName) != "" {
		return strings.TrimSpace(p.DisplayName)
	}
	return "Pilot"
}

// Build is a curated or temporary parts list.
type Build struct {
	ID               string      `json:"id"`
	OwnerUserID      string      `json:"ownerUserId,omitempty"`
	ImageAssetID     string      `json:"-"`
	Status           BuildStatus `json:"status"`
	Token            string      `json:"-"`
	ExpiresAt        *time.Time  `json:"expiresAt,omitempty"`
	Title            string      `json:"title"`
	Description      string      `json:"description,omitempty"`
	SourceAircraftID string      `json:"sourceAircraftId,omitempty"`
	CreatedAt        time.Time   `json:"createdAt"`
	UpdatedAt        time.Time   `json:"updatedAt"`
	PublishedAt      *time.Time  `json:"publishedAt,omitempty"`
	Parts            []BuildPart `json:"parts,omitempty"`
	Verified         bool        `json:"verified"`
	MainImageURL     string      `json:"mainImageUrl,omitempty"`
	Pilot            *BuildPilot `json:"pilot,omitempty"`
}

// CreateBuildParams defines payload for new authenticated builds.
type CreateBuildParams struct {
	Title            string           `json:"title"`
	Description      string           `json:"description,omitempty"`
	SourceAircraftID string           `json:"sourceAircraftId,omitempty"`
	Parts            []BuildPartInput `json:"parts,omitempty"`
}

// UpdateBuildParams defines payload for editing a build.
type UpdateBuildParams struct {
	Title       *string          `json:"title,omitempty"`
	Description *string          `json:"description,omitempty"`
	Parts       []BuildPartInput `json:"parts,omitempty"`
}

// SetBuildImageParams defines parameters for uploading a build image.
type SetBuildImageParams struct {
	BuildID   string
	ImageType string // "image/jpeg", "image/png", or "image/webp"
	ImageData []byte
	UploadID  string // approved token from /api/images/upload
}

// BuildListParams describes list query options.
type BuildListParams struct {
	Sort        BuildSort `json:"sort,omitempty"`
	FrameFilter string    `json:"frameFilter,omitempty"`
	Limit       int       `json:"limit,omitempty"`
	Offset      int       `json:"offset,omitempty"`
}

// BuildModerationListParams describes admin moderation list query options.
type BuildModerationListParams struct {
	Query  string      `json:"query,omitempty"`
	Status BuildStatus `json:"status,omitempty"`
	Limit  int         `json:"limit,omitempty"`
	Offset int         `json:"offset,omitempty"`
}

// BuildListResponse is returned by build list endpoints.
type BuildListResponse struct {
	Builds      []Build   `json:"builds"`
	TotalCount  int       `json:"totalCount"`
	Sort        BuildSort `json:"sort,omitempty"`
	FrameFilter string    `json:"frameFilter,omitempty"`
}

// BuildValidationError is a single publish validation issue.
type BuildValidationError struct {
	Category string `json:"category"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

// BuildValidationResult captures server publish validation output.
type BuildValidationResult struct {
	Valid  bool                   `json:"valid"`
	Errors []BuildValidationError `json:"errors,omitempty"`
}

// BuildPublishResponse includes updated build and validation status.
type BuildPublishResponse struct {
	Build      *Build                `json:"build,omitempty"`
	Validation BuildValidationResult `json:"validation"`
}

// TempBuildCreateResponse is returned after creating a temporary build.
type TempBuildCreateResponse struct {
	Build *Build `json:"build"`
	Token string `json:"token"`
	URL   string `json:"url"`
}
