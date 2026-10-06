package testsupport

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/edoazn/absensi-go/config"
	"github.com/edoazn/absensi-go/internal/database"
	"github.com/edoazn/absensi-go/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const TestPassword = "rahasia123"

func TestDBName() string {
	return fmt.Sprintf("absensi_go_test_%d", os.Getpid())
}

func TestConfig(t *testing.T) *config.Config {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}
	return &config.Config{
		AppName:         "Absensi Test",
		AppEnv:          "testing",
		Port:            "0",
		DBHost:          "127.0.0.1",
		DBPort:          "3306",
		DBUser:          "root",
		DBPassword:      "root",
		DBName:          TestDBName(),
		JWTSecret:       "test-secret-0123456789abcdef0123456789abcdef",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 720 * time.Hour,
		BcryptCost:      bcrypt.MinCost,
		Timezone:        loc,
	}
}

func SetupDB(t *testing.T) (*config.Config, *gorm.DB) {
	t.Helper()
	cfg := TestConfig(t)

	if err := database.EnsureDatabase(cfg); err != nil {
		t.Skipf("mysql not available, skipping integration test: %v", err)
	}
	if err := database.Migrate(cfg); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db, err := database.Connect(cfg)
	if err != nil {
		t.Skipf("mysql not available, skipping integration test: %v", err)
	}
	TruncateAll(t, db)
	return cfg, db
}

func TruncateAll(t *testing.T, db *gorm.DB) {
	t.Helper()
	tables := []string{
		"attendances", "refresh_tokens", "schedules", "courses",
		"locations", "class_user", "classes", "users",
	}
	if err := db.Exec("SET FOREIGN_KEY_CHECKS = 0").Error; err != nil {
		t.Fatalf("disable FK checks: %v", err)
	}
	for _, tb := range tables {
		if err := db.Exec("TRUNCATE TABLE " + tb).Error; err != nil {
			t.Fatalf("truncate %s: %v", tb, err)
		}
	}
	if err := db.Exec("SET FOREIGN_KEY_CHECKS = 1").Error; err != nil {
		t.Fatalf("enable FK checks: %v", err)
	}
}

func CreateUser(t *testing.T, db *gorm.DB, name, identityNumber, role string, email *string) models.User {
	t.Helper()
	hashed, err := bcrypt.GenerateFromPassword([]byte(TestPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := models.User{
		Name:           name,
		IdentityNumber: identityNumber,
		Email:          email,
		Password:       string(hashed),
		Role:           role,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user %s: %v", identityNumber, err)
	}
	return user
}

func PtrString(s string) *string  { return &s }
func PtrFloat(f float64) *float64 { return &f }
