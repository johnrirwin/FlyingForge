package moderation

import (
	"context"

	"github.com/johnrirwin/flyingforge/internal/models"
)

// MockModerator is a simple mock implementation for tests.
type MockModerator struct {
	Decision *models.ModerationDecision
	Err      error
}

// ModerateImageBytes returns the configured decision/error.
func (m *MockModerator) ModerateImageBytes(ctx context.Context, imageBytes []byte) (*models.ModerationDecision, error) {
	_ = ctx
	_ = imageBytes
	if m.Err != nil {
		return nil, m.Err
	}
	if m.Decision != nil {
		return m.Decision, nil
	}
	return &models.ModerationDecision{
		Status: models.ImageModerationApproved,
		Reason: "Approved",
	}, nil
}
