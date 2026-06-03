package config

import (
	"strings"
	"testing"
)

// setRequired sets the minimum env for a successful Load; individual tests
// override what they exercise.
func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/kashmir")
	t.Setenv("JWT_SECRET", "test-secret")
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "test-secret")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("err = %v, want DATABASE_URL required", err)
	}
}

func TestLoadRequiresJWTSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/kashmir")
	t.Setenv("JWT_SECRET", "")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("err = %v, want JWT_SECRET required", err)
	}
}

func TestLoadDefaults(t *testing.T) {
	setRequired(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.Env != "development" {
		t.Errorf("Env = %q, want development", cfg.Env)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
}

func TestLoadDevBypassAllowedInDevelopment(t *testing.T) {
	setRequired(t)
	t.Setenv("ENV", "development")
	t.Setenv("OTP_DEV_BYPASS", "true")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.OTP.DevBypass {
		t.Error("expected DevBypass=true in development")
	}
}

func TestLoadDevBypassRejectedOutsideDevelopment(t *testing.T) {
	for _, env := range []string{"production", "staging"} {
		t.Run(env, func(t *testing.T) {
			setRequired(t)
			t.Setenv("ENV", env)
			t.Setenv("OTP_DEV_BYPASS", "true")
			if _, err := Load(); err == nil || !strings.Contains(err.Error(), "OTP_DEV_BYPASS") {
				t.Fatalf("err = %v, want OTP_DEV_BYPASS rejected in %s", err, env)
			}
		})
	}
}
