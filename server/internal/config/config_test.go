package config

import (
	"flag"
	"io"
	"os"
	"testing"
	"time"
)

func loadWithArgs(t *testing.T, args ...string) *Config {
	t.Helper()

	if len(args) == 0 {
		args = []string{"test"}
	}

	oldCommandLine := flag.CommandLine
	oldArgs := os.Args

	flag.CommandLine = flag.NewFlagSet(args[0], flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)
	os.Args = args

	t.Cleanup(func() {
		flag.CommandLine = oldCommandLine
		os.Args = oldArgs
	})

	return Load()
}

func TestLoad_EnableManualRefresh_FromEnv(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		t.Setenv("ENABLE_MANUAL_REFRESH", "true")
		cfg := loadWithArgs(t, "test")
		if !cfg.Server.EnableManualRefresh {
			t.Fatalf("expected EnableManualRefresh=true when ENABLE_MANUAL_REFRESH=true")
		}
	})

	t.Run("one", func(t *testing.T) {
		t.Setenv("ENABLE_MANUAL_REFRESH", "1")
		cfg := loadWithArgs(t, "test")
		if !cfg.Server.EnableManualRefresh {
			t.Fatalf("expected EnableManualRefresh=true when ENABLE_MANUAL_REFRESH=1")
		}
	})

	t.Run("false", func(t *testing.T) {
		t.Setenv("ENABLE_MANUAL_REFRESH", "false")
		cfg := loadWithArgs(t, "test")
		if cfg.Server.EnableManualRefresh {
			t.Fatalf("expected EnableManualRefresh=false when ENABLE_MANUAL_REFRESH=false")
		}
	})
}

func TestLoad_RefreshOnceMode_FromEnv(t *testing.T) {
	t.Setenv("REFRESH_ONCE_MODE", "true")
	cfg := loadWithArgs(t, "test")
	if !cfg.Server.RefreshOnceMode {
		t.Fatalf("expected RefreshOnceMode=true when REFRESH_ONCE_MODE=true")
	}
}

func TestLoad_RefreshOnceMode_FromFlag(t *testing.T) {
	t.Setenv("REFRESH_ONCE_MODE", "")
	cfg := loadWithArgs(t, "test", "-refresh-once")
	if !cfg.Server.RefreshOnceMode {
		t.Fatalf("expected RefreshOnceMode=true when -refresh-once is provided")
	}
}

func TestLoadMCPConfig_AuthEnabledRequiresIssuer(t *testing.T) {
	tests := []struct {
		name         string
		issuer       string
		discoveryURL string
		jwksURL      string
		wantEnabled  bool
	}{
		{
			name:         "disabled when only discovery url is set",
			discoveryURL: "https://issuer.example/.well-known/openid-configuration",
			wantEnabled:  false,
		},
		{
			name:        "disabled when only jwks url is set",
			jwksURL:     "https://issuer.example/.well-known/jwks.json",
			wantEnabled: false,
		},
		{
			name:         "enabled when issuer is set",
			issuer:       "https://issuer.example",
			discoveryURL: "https://issuer.example/.well-known/openid-configuration",
			jwksURL:      "https://issuer.example/.well-known/jwks.json",
			wantEnabled:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MCP_AUTH_ISSUER", tt.issuer)
			t.Setenv("MCP_AUTH_DISCOVERY_URL", tt.discoveryURL)
			t.Setenv("MCP_AUTH_JWKS_URL", tt.jwksURL)

			cfg := loadMCPConfig()
			if cfg.Auth.Enabled != tt.wantEnabled {
				t.Fatalf("expected Auth.Enabled=%t, got %t", tt.wantEnabled, cfg.Auth.Enabled)
			}
		})
	}
}

func TestLoadMCPConfig_AllowedOriginsDefaultsWhenConfiguredListIsEffectivelyEmpty(t *testing.T) {
	t.Setenv("MCP_ALLOWED_ORIGINS", ",")

	cfg := loadMCPConfig()

	want := []string{
		"https://chatgpt.com",
		"https://chat.openai.com",
	}
	if len(cfg.AllowedOrigins) != len(want) {
		t.Fatalf("expected %d default allowed origins, got %d (%v)", len(want), len(cfg.AllowedOrigins), cfg.AllowedOrigins)
	}
	for i, expected := range want {
		if cfg.AllowedOrigins[i] != expected {
			t.Fatalf("expected default allowed origin %q at index %d, got %q", expected, i, cfg.AllowedOrigins[i])
		}
	}
}

func TestLoadMCPConfig_AllowedOriginsUsesConfiguredListWhenNonEmpty(t *testing.T) {
	t.Setenv("MCP_ALLOWED_ORIGINS", "https://example.com, https://chatgpt.com")

	cfg := loadMCPConfig()

	want := []string{"https://example.com", "https://chatgpt.com"}
	if len(cfg.AllowedOrigins) != len(want) {
		t.Fatalf("expected %d allowed origins, got %d (%v)", len(want), len(cfg.AllowedOrigins), cfg.AllowedOrigins)
	}
	for i, expected := range want {
		if cfg.AllowedOrigins[i] != expected {
			t.Fatalf("expected allowed origin %q at index %d, got %q", expected, i, cfg.AllowedOrigins[i])
		}
	}
}

func TestLoadMCPConfig_SelfHostedDefaultsFromPublicBaseURL(t *testing.T) {
	t.Setenv("MCP_PUBLIC_BASE_URL", "https://flyingforge.example")
	t.Setenv("MCP_AUTH_SELF_HOSTED", "true")

	cfg := loadMCPConfig()

	if !cfg.Auth.SelfHosted {
		t.Fatalf("expected self-hosted auth to be enabled")
	}
	if cfg.Auth.Issuer != "https://flyingforge.example" {
		t.Fatalf("expected issuer to default from public base URL, got %q", cfg.Auth.Issuer)
	}
	if cfg.Auth.GoogleRedirectURI != "https://flyingforge.example/oauth/google/callback" {
		t.Fatalf("expected Google redirect URI to default from public base URL, got %q", cfg.Auth.GoogleRedirectURI)
	}
	if cfg.Auth.AccessTokenTTL != time.Hour {
		t.Fatalf("expected default access token TTL of 1h, got %s", cfg.Auth.AccessTokenTTL)
	}
	if cfg.Auth.AuthorizationCodeTTL != 10*time.Minute {
		t.Fatalf("expected default auth code TTL of 10m, got %s", cfg.Auth.AuthorizationCodeTTL)
	}
	if cfg.Auth.RefreshTokenTTL != 30*24*time.Hour {
		t.Fatalf("expected default refresh token TTL of 30d, got %s", cfg.Auth.RefreshTokenTTL)
	}
	if cfg.Auth.SessionTTL != 24*time.Hour {
		t.Fatalf("expected default session TTL of 24h, got %s", cfg.Auth.SessionTTL)
	}
}

func TestLoadMCPConfig_SelfHostedDurationOverrides(t *testing.T) {
	t.Setenv("MCP_AUTH_SELF_HOSTED", "true")
	t.Setenv("MCP_AUTH_ISSUER", "https://issuer.example")
	t.Setenv("MCP_AUTH_GOOGLE_REDIRECT_URI", "https://issuer.example/custom-google-callback")
	t.Setenv("MCP_AUTH_ACCESS_TOKEN_TTL", "2h")
	t.Setenv("MCP_AUTH_CODE_TTL", "15m")
	t.Setenv("MCP_AUTH_REFRESH_TOKEN_TTL", "720h")
	t.Setenv("MCP_AUTH_SESSION_TTL", "12h")

	cfg := loadMCPConfig()

	if cfg.Auth.GoogleRedirectURI != "https://issuer.example/custom-google-callback" {
		t.Fatalf("expected explicit Google redirect URI override, got %q", cfg.Auth.GoogleRedirectURI)
	}
	if cfg.Auth.AccessTokenTTL != 2*time.Hour {
		t.Fatalf("expected access token TTL override, got %s", cfg.Auth.AccessTokenTTL)
	}
	if cfg.Auth.AuthorizationCodeTTL != 15*time.Minute {
		t.Fatalf("expected auth code TTL override, got %s", cfg.Auth.AuthorizationCodeTTL)
	}
	if cfg.Auth.RefreshTokenTTL != 720*time.Hour {
		t.Fatalf("expected refresh token TTL override, got %s", cfg.Auth.RefreshTokenTTL)
	}
	if cfg.Auth.SessionTTL != 12*time.Hour {
		t.Fatalf("expected session TTL override, got %s", cfg.Auth.SessionTTL)
	}
}
