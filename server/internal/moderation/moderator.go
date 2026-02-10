package moderation

import (
	"context"

	"github.com/johnrirwin/flyingforge/internal/models"
)

// Detector is the low-level provider abstraction that fetches moderation labels.
type Detector interface {
	DetectModerationLabels(ctx context.Context, imageBytes []byte) ([]models.ModerationLabel, error)
}

// Service evaluates moderation labels into APPROVED/REJECTED decisions.
type Service struct {
	detector         Detector
	rejectConfidence float64
}

// NewService creates a moderation service using the configured detector.
func NewService(detector Detector, rejectConfidence float64) *Service {
	if rejectConfidence <= 0 {
		rejectConfidence = 70
	}
	return &Service{
		detector:         detector,
		rejectConfidence: rejectConfidence,
	}
}

// ModerateImageBytes moderates image bytes and returns an APPROVED/REJECTED decision.
func (s *Service) ModerateImageBytes(ctx context.Context, imageBytes []byte) (*models.ModerationDecision, error) {
	labels, err := s.detector.DetectModerationLabels(ctx, imageBytes)
	if err != nil {
		return nil, err
	}

	decision := &models.ModerationDecision{
		Status: models.ImageModerationApproved,
		Reason: "Approved",
		Labels: labels,
	}

	maxConfidence := 0.0
	shouldReject := false
	for _, label := range labels {
		if label.Confidence > maxConfidence {
			maxConfidence = label.Confidence
		}
		if label.Confidence >= s.rejectConfidence {
			shouldReject = true
		}
	}
	decision.MaxConfidence = maxConfidence

	if shouldReject {
		decision.Status = models.ImageModerationRejected
		decision.Reason = "Not allowed"
	}

	return decision, nil
}
