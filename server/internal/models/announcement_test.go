package models

import "testing"

func TestIsValidAnnouncementCTAURL(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "empty is allowed", value: "", want: true},
		{name: "relative path", value: "/news", want: true},
		{name: "relative path with query", value: "/news?tab=latest", want: true},
		{name: "absolute https url", value: "https://example.com/news", want: true},
		{name: "scheme relative url is rejected", value: "//evil.com", want: false},
		{name: "triple slash url is rejected", value: "///evil.com", want: false},
		{name: "unsupported scheme", value: "javascript:alert(1)", want: false},
		{name: "missing host", value: "https:/broken", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidAnnouncementCTAURL(tt.value); got != tt.want {
				t.Fatalf("IsValidAnnouncementCTAURL(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}
