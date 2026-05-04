package config

import (
	"flag"
	"io"
	"os"
	"testing"
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
