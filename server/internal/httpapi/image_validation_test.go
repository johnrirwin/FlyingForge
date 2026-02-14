package httpapi

import "testing"

func TestDetectAllowedImageContentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       []byte
		wantType    string
		wantAllowed bool
	}{
		{
			name:        "empty",
			input:       nil,
			wantType:    "",
			wantAllowed: false,
		},
		{
			name:        "jpeg_allowed",
			input:       append([]byte{0xFF, 0xD8, 0xFF, 0xDB}, make([]byte, 512)...),
			wantType:    "image/jpeg",
			wantAllowed: true,
		},
		{
			name:        "png_allowed",
			input:       append([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, make([]byte, 512)...),
			wantType:    "image/png",
			wantAllowed: true,
		},
		{
			name:        "webp_disallowed",
			input:       append([]byte("RIFF\x00\x00\x00\x00WEBPVP8 "), make([]byte, 512)...),
			wantType:    "image/webp",
			wantAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotType, gotAllowed := detectAllowedImageContentType(tt.input)
			if gotType != tt.wantType {
				t.Fatalf("contentType=%q, want %q", gotType, tt.wantType)
			}
			if gotAllowed != tt.wantAllowed {
				t.Fatalf("allowed=%v, want %v", gotAllowed, tt.wantAllowed)
			}
		})
	}
}
