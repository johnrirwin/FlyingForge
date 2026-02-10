package moderation

import (
	"context"
	"errors"
	"testing"

	"github.com/johnrirwin/flyingforge/internal/models"
)

type fakeDetector struct {
	labels []models.ModerationLabel
	err    error
}

func (f *fakeDetector) DetectModerationLabels(ctx context.Context, imageBytes []byte) ([]models.ModerationLabel, error) {
	_ = ctx
	_ = imageBytes
	if f.err != nil {
		return nil, f.err
	}
	return f.labels, nil
}

func TestServiceModerateImageBytes(t *testing.T) {
	tests := []struct {
		name       string
		labels     []models.ModerationLabel
		err        error
		threshold  float64
		wantStatus models.ImageModerationStatus
		wantMax    float64
	}{
		{
			name:       "approved when no labels",
			labels:     nil,
			threshold:  70,
			wantStatus: models.ImageModerationApproved,
			wantMax:    0,
		},
		{
			name: "approved when labels below threshold",
			labels: []models.ModerationLabel{
				{Name: "Suggestive", Confidence: 42.1},
			},
			threshold:  70,
			wantStatus: models.ImageModerationApproved,
			wantMax:    42.1,
		},
		{
			name: "rejected when any label meets threshold",
			labels: []models.ModerationLabel{
				{Name: "Explicit Nudity", Confidence: 82.3},
				{Name: "Violence", Confidence: 50.0},
			},
			threshold:  70,
			wantStatus: models.ImageModerationRejected,
			wantMax:    82.3,
		},
		{
			name:      "returns detector errors",
			err:       errors.New("boom"),
			threshold: 70,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&fakeDetector{
				labels: tt.labels,
				err:    tt.err,
			}, tt.threshold)

			decision, err := svc.ModerateImageBytes(context.Background(), []byte("abc"))
			if tt.err != nil {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if decision.Status != tt.wantStatus {
				t.Fatalf("status=%s want=%s", decision.Status, tt.wantStatus)
			}
			if decision.MaxConfidence != tt.wantMax {
				t.Fatalf("max=%v want=%v", decision.MaxConfidence, tt.wantMax)
			}
		})
	}
}
