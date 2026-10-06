package config

import (
	"strings"
	"testing"
	"time"
)

const validSecret = "0123456789abcdef0123456789abcdef"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("JWT_SECRET", validSecret)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.DBName != "absensi_go" {
		t.Errorf("DBName = %q, want absensi_go", cfg.DBName)
	}
	if cfg.AccessTokenTTL != 15*time.Minute {
		t.Errorf("AccessTokenTTL = %v, want 15m", cfg.AccessTokenTTL)
	}
	if cfg.RefreshTokenTTL != 720*time.Hour {
		t.Errorf("RefreshTokenTTL = %v, want 720h", cfg.RefreshTokenTTL)
	}
	if cfg.BcryptCost != 12 {
		t.Errorf("BcryptCost = %d, want 12", cfg.BcryptCost)
	}
	if cfg.Timezone.String() != "Asia/Jakarta" {
		t.Errorf("Timezone = %q, want Asia/Jakarta", cfg.Timezone.String())
	}
	if cfg.SeedOnStart {
		t.Error("SeedOnStart default must be false")
	}
}

func TestLoadDSN(t *testing.T) {
	t.Setenv("JWT_SECRET", validSecret)
	t.Setenv("DB_HOST", "10.0.0.5")
	t.Setenv("DB_PORT", "3307")
	t.Setenv("DB_USERNAME", "app")
	t.Setenv("DB_PASSWORD", "s3cret")
	t.Setenv("DB_DATABASE", "absensi_prod")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	dsn := cfg.MySQLDSN()
	for _, want := range []string{
		"app:s3cret@tcp(10.0.0.5:3307)/absensi_prod",
		"parseTime=True",
		"loc=Asia%2FJakarta",
	} {
		if !strings.Contains(dsn, want) {
			t.Errorf("MySQLDSN missing %q in %q", want, dsn)
		}
	}

	server := cfg.MySQLServerDSN(true)
	if !strings.Contains(server, "multiStatements=true") {
		t.Errorf("MySQLServerDSN(true) must contain multiStatements=true: %q", server)
	}
	if strings.Contains(server, "/absensi_prod") {
		t.Errorf("server DSN must not select a database: %q", server)
	}
}

func TestLoadRejectsShortSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "too-short")

	if _, err := Load(); err == nil {
		t.Fatal("expected error for short JWT_SECRET")
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	t.Setenv("JWT_SECRET", validSecret)
	t.Setenv("JWT_ACCESS_TTL", "bukan-duration")

	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid JWT_ACCESS_TTL")
	}
}

func TestLoadInvalidTimezone(t *testing.T) {
	t.Setenv("JWT_SECRET", validSecret)
	t.Setenv("APP_TIMEZONE", "Bumi/Mars")

	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid APP_TIMEZONE")
	}
}
