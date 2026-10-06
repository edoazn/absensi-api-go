package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/edoazn/absensi-go/config"
	"github.com/edoazn/absensi-go/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const issuerName = "absensi-go"

type AccessTokenClaims struct {
	UserID uint   `json:"uid"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type TokenService struct {
	cfg *config.Config
	db  *gorm.DB
}

func NewTokenService(cfg *config.Config, db *gorm.DB) *TokenService {
	return &TokenService{cfg: cfg, db: db}
}

func (s *TokenService) TTLSeconds() int64 {
	return int64(s.cfg.AccessTokenTTL.Seconds())
}

func (s *TokenService) GenerateAccessToken(user *models.User) (raw, jti string, err error) {
	now := time.Now()
	jti = uuid.NewString()
	claims := AccessTokenClaims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Issuer:    issuerName,
			Subject:   fmt.Sprintf("%d", user.ID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.AccessTokenTTL)),
		},
	}
	signed, signErr := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.JWTSecret))
	if signErr != nil {
		return "", "", fmt.Errorf("sign access token: %w", signErr)
	}
	return signed, jti, nil
}

func (s *TokenService) ParseAccessToken(raw string) (*AccessTokenClaims, error) {
	claims := &AccessTokenClaims{}
	token, err := jwt.ParseWithClaims(
		raw,
		claims,
		func(t *jwt.Token) (any, error) { return []byte(s.cfg.JWTSecret), nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuerName),
	)
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid access token")
	}
	return claims, nil
}

func (s *TokenService) IssueRefreshToken(userID uint) (string, error) {
	raw, err := randomToken()
	if err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	row := models.RefreshToken{
		UserID:    userID,
		TokenHash: sha256Hex(raw),
		ExpiresAt: time.Now().Add(s.cfg.RefreshTokenTTL),
	}
	if err := s.db.Create(&row).Error; err != nil {
		return "", fmt.Errorf("store refresh token: %w", err)
	}
	return raw, nil
}

func (s *TokenService) LookupActiveRefreshToken(raw string) (*models.RefreshToken, error) {
	var row models.RefreshToken
	err := s.db.Preload("User").
		Where("token_hash = ?", sha256Hex(raw)).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	if !row.IsActive(time.Now()) {
		return nil, fmt.Errorf("refresh token inactive")
	}
	return &row, nil
}

func (s *TokenService) RotateRefreshToken(old *models.RefreshToken) (string, error) {
	newRaw, err := randomToken()
	if err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	newHash := sha256Hex(newRaw)

	err = s.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		res := tx.Model(&models.RefreshToken{}).
			Where("id = ? AND revoked_at IS NULL", old.ID).
			Update("revoked_at", now)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("refresh token already rotated")
		}
		return tx.Create(&models.RefreshToken{
			UserID:    old.UserID,
			TokenHash: newHash,
			ExpiresAt: now.Add(s.cfg.RefreshTokenTTL),
		}).Error
	})
	if err != nil {
		return "", err
	}
	return newRaw, nil
}

func (s *TokenService) RevokeByRawToken(raw string) error {
	return s.db.Model(&models.RefreshToken{}).
		Where("token_hash = ? AND revoked_at IS NULL", sha256Hex(raw)).
		Update("revoked_at", time.Now()).Error
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func sha256Hex(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
