package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/johnrirwin/flyingforge/internal/moderation"
)

func main() {
	defaultImage := os.Getenv("IMAGE")
	imagePath := flag.String("image", defaultImage, "path to local image file")
	flag.Parse()

	if *imagePath == "" {
		fmt.Fprintln(os.Stderr, "image path is required (pass -image or IMAGE env var)")
		os.Exit(1)
	}

	imageBytes, err := os.ReadFile(*imagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read image: %v\n", err)
		os.Exit(1)
	}

	region := os.Getenv("AWS_REGION")
	detector, err := moderation.NewAWSDetector(context.Background(), region)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize rekognition detector: %v\n", err)
		os.Exit(1)
	}

	rejectConfidence := 70.0
	if raw := os.Getenv("MODERATION_REJECT_CONFIDENCE"); raw != "" {
		if parsed, err := strconv.ParseFloat(raw, 64); err == nil && parsed > 0 {
			rejectConfidence = parsed
		}
	}
	moderator := moderation.NewService(detector, rejectConfidence)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	decision, err := moderator.ModerateImageBytes(ctx, imageBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rekognition call failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Status: %s\n", decision.Status)
	if decision.Reason != "" {
		fmt.Printf("Reason: %s\n", decision.Reason)
	}
	fmt.Printf("MaxConfidence: %.2f\n", decision.MaxConfidence)
	fmt.Println("Labels:")
	for _, label := range decision.Labels {
		if label.ParentName != "" {
			fmt.Printf("  - %s (%s): %.2f\n", label.Name, label.ParentName, label.Confidence)
		} else {
			fmt.Printf("  - %s: %.2f\n", label.Name, label.Confidence)
		}
	}
}
