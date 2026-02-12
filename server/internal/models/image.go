package models

import (
	"encoding/json"
	"time"
)

// ImageEntityType identifies what record an image belongs to.
type ImageEntityType string

const (
	ImageEntityAvatar   ImageEntityType = "avatar"
	ImageEntityAircraft ImageEntityType = "aircraft"
	ImageEntityBuild    ImageEntityType = "build"
	ImageEntityGear     ImageEntityType = "gear"
	ImageEntityOther    ImageEntityType = "other"
)

// ImageModerationStatus is the moderation outcome returned to clients.
type ImageModerationStatus string

const (
	ImageModerationApproved      ImageModerationStatus = "APPROVED"
	ImageModerationRejected      ImageModerationStatus = "REJECTED"
	ImageModerationPendingReview ImageModerationStatus = "PENDING_REVIEW"
)

// ModerationLabel captures a single Rekognition moderation label.
type ModerationLabel struct {
	Name       string  `json:"name"`
	ParentName string  `json:"parentName,omitempty"`
	Confidence float64 `json:"confidence"`
}

// ModerationDecision is the server-side decision used by upload flows.
type ModerationDecision struct {
	Status        ImageModerationStatus `json:"status"`
	Reason        string                `json:"reason,omitempty"`
	Labels        []ModerationLabel     `json:"labels,omitempty"`
	MaxConfidence float64               `json:"maxConfidence,omitempty"`
}

// ImageAsset stores approved image bytes + moderation metadata.
type ImageAsset struct {
	ID                      string
	OwnerUserID             string
	EntityType              ImageEntityType
	EntityID                string
	ImageBytes              []byte
	Status                  ImageModerationStatus
	ModerationLabels        json.RawMessage
	ModerationMaxConfidence float64
	CreatedAt               time.Time
	UpdatedAt               time.Time
}
