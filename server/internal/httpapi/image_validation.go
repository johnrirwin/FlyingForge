package httpapi

import (
	"net/http"
	"strings"
)

var allowedImageContentTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
}

func detectAllowedImageContentType(imageData []byte) (string, bool) {
	if len(imageData) == 0 {
		return "", false
	}

	contentType := strings.ToLower(strings.TrimSpace(http.DetectContentType(imageData)))
	if contentType == "image/jpg" {
		contentType = "image/jpeg"
	}

	_, ok := allowedImageContentTypes[contentType]
	return contentType, ok
}
