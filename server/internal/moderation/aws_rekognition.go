package moderation

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	rekognitiontypes "github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/johnrirwin/flyingforge/internal/models"
)

// AWSDetector calls Rekognition using AWS SDK with byte payloads (no S3 dependency).
type AWSDetector struct {
	client *rekognition.Client
}

// NewAWSDetector creates a detector that uses ambient AWS credentials/profile.
func NewAWSDetector(ctx context.Context, region string) (*AWSDetector, error) {
	loadOptions := []func(*awsconfig.LoadOptions) error{}
	trimmedRegion := strings.TrimSpace(region)
	if trimmedRegion != "" {
		loadOptions = append(loadOptions, awsconfig.WithRegion(trimmedRegion))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	return &AWSDetector{
		client: rekognition.NewFromConfig(cfg),
	}, nil
}

// DetectModerationLabels calls Rekognition DetectModerationLabels with raw image bytes.
func (d *AWSDetector) DetectModerationLabels(ctx context.Context, imageBytes []byte) ([]models.ModerationLabel, error) {
	if len(imageBytes) == 0 {
		return nil, fmt.Errorf("image bytes are required")
	}

	output, err := d.client.DetectModerationLabels(ctx, &rekognition.DetectModerationLabelsInput{
		Image: &rekognitiontypes.Image{
			Bytes: imageBytes,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("rekognition detect moderation labels failed: %w", err)
	}

	labels := make([]models.ModerationLabel, 0, len(output.ModerationLabels))
	for _, label := range output.ModerationLabels {
		confidence := 0.0
		if label.Confidence != nil {
			confidence = float64(*label.Confidence)
		}

		labels = append(labels, models.ModerationLabel{
			Name:       aws.ToString(label.Name),
			ParentName: aws.ToString(label.ParentName),
			Confidence: confidence,
		})
	}

	return labels, nil
}
