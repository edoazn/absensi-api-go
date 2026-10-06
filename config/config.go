package config

import (
	_ "time/tzdata"

	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName         string
	AppEnv          string
	Port            string
	DBHost          string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	BcryptCost      int
	Timezone        *time.Location
	SeedOnStart     bool
	CorsOrigins     []string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	locName := getEnv("APP_TIMEZONE", "Asia/Jakarta")
	loc, err := time.LoadLocation(locName)
	if err != nil {
		return nil, fmt.Errorf("invalid APP_TIMEZONE %q: %w", locName, err)
	}

	accessTTL, err := time.ParseDuration(getEnv("JWT_ACCESS_TTL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_TTL: %w", err)
	}

	refreshTTL, err := time.ParseDuration(getEnv("JWT_REFRESH_TTL", "720h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_TTL: %w", err)
	}

	cost := 12
	if v := os.Getenv("BCRYPT_COST"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &cost); err != nil {
			return nil, fmt.Errorf("invalid BCRYPT_COST: %w", err)
		}
	}

	cfg := &Config{
		AppName:         getEnv("APP_NAME", "Absensi Mahasiswa API"),
		AppEnv:          getEnv("APP_ENV", "local"),
		Port:            getEnv("APP_PORT", "8080"),
		DBHost:          getEnv("DB_HOST", "127.0.0.1"),
		DBPort:          getEnv("DB_PORT", "3306"),
		DBUser:          getEnv("DB_USERNAME", "root"),
		DBPassword:      os.Getenv("DB_PASSWORD"),
		DBName:          getEnv("DB_DATABASE", "absensi_go"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,
		BcryptCost:      cost,
		Timezone:        loc,
		SeedOnStart:     getEnv("SEED_ON_START", "false") == "true",
		CorsOrigins:     parseOrigins(os.Getenv("CORS_ORIGINS")),
	}

	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}

	return cfg, nil
}

func (c *Config) IsProd() bool {
	return c.AppEnv == "production"
}

func (c *Config) MySQLDSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
		url.QueryEscape(c.Timezone.String()),
	)
}

func (c *Config) MySQLServerDSN(multiStatements bool) string {
	extra := ""
	if multiStatements {
		extra = "&multiStatements=true"
	}
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=%s%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort,
		url.QueryEscape(c.Timezone.String()), extra,
	)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// parseOrigins memecah daftar origin CORS dari env dipisah koma.
// Kosong berarti CORS dimatikan; "*" mengizinkan semua origin.
func parseOrigins(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
