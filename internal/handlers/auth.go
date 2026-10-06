package handlers

import (
	"errors"
	"net/http"

	"github.com/edoazn/absensi-go/internal/api"
	"github.com/edoazn/absensi-go/internal/dto"
	"github.com/edoazn/absensi-go/internal/middleware"
	"github.com/edoazn/absensi-go/internal/models"
	"github.com/edoazn/absensi-go/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db        *gorm.DB
	tokens    *services.TokenService
	blacklist *services.Blacklist
}

func NewAuthHandler(db *gorm.DB, tokens *services.TokenService, blacklist *services.Blacklist) *AuthHandler {
	return &AuthHandler{db: db, tokens: tokens, blacklist: blacklist}
}

// Login Login dengan NIM/NIP
// @Summary Login dengan NIM/NIP
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body dto.LoginRequest true "Kredensial"
// @Success 200 {object} api.Envelope
// @Failure 401 {object} api.Envelope
// @Failure 422 {object} api.Envelope
// @Failure 429 {object} api.Envelope
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if !api.BindJSON(c, &req) {
		return
	}

	var user models.User
	err := h.db.Where("identity_number = ?", req.IdentityNumber).First(&user).Error
	if err != nil || !services.CheckPassword(user.Password, req.Password) {
		api.Fail(c, http.StatusUnauthorized, "NIM/NIP atau password salah")
		return
	}

	data, err := h.issueSession(&user)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal membuat sesi login")
		return
	}
	api.OK(c, data, "Login berhasil")
}

// Refresh Rotasi refresh token
// @Summary Rotasi refresh token
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body dto.RefreshRequest true "Refresh token"
// @Success 200 {object} api.Envelope
// @Failure 401 {object} api.Envelope
// @Router /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.RefreshRequest
	if !api.BindJSON(c, &req) {
		return
	}

	row, err := h.tokens.LookupActiveRefreshToken(req.RefreshToken)
	if err != nil {
		api.Fail(c, http.StatusUnauthorized, "Refresh token tidak valid")
		return
	}

	newRaw, err := h.tokens.RotateRefreshToken(row)
	if err != nil {
		api.Fail(c, http.StatusUnauthorized, "Refresh token sudah digunakan")
		return
	}

	access, _, err := h.tokens.GenerateAccessToken(&row.User)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal membuat access token")
		return
	}

	data := dto.AuthData{
		AccessToken:  access,
		RefreshToken: newRaw,
		TokenType:    "Bearer",
		ExpiresIn:    h.tokens.TTLSeconds(),
		User:         dto.NewUserProfile(&row.User),
	}
	api.OK(c, data, "Token berhasil diperbarui")
}

// Logout Logout (revoke refresh + blacklist access)
// @Summary Logout (revoke refresh + blacklist access)
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body dto.RefreshRequest false "Refresh token"
// @Success 200 {object} api.Envelope
// @Failure 401 {object} api.Envelope
// @Security BearerAuth
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	if claims := middleware.ClaimsFrom(c); claims != nil {
		h.blacklist.Add(claims.ID, claims.ExpiresAt.Time)
	}

	var req dto.RefreshRequest
	if err := c.ShouldBindJSON(&req); err == nil && req.RefreshToken != "" {
		_ = h.tokens.RevokeByRawToken(req.RefreshToken)
	}

	api.OK(c, nil, "Logout berhasil")
}

// Me Profil user login
// @Summary Profil user login
// @Tags Auth
// @Accept json
// @Produce json
// @Success 200 {object} api.Envelope
// @Failure 401 {object} api.Envelope
// @Security BearerAuth
// @Router /me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	var user models.User
	if err := h.db.First(&user, middleware.UserIDFrom(c)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.Fail(c, http.StatusUnauthorized, "Unauthenticated")
			return
		}
		api.Fail(c, http.StatusInternalServerError, "Gagal memuat profil")
		return
	}
	api.OK(c, dto.NewUserProfile(&user), "")
}

func (h *AuthHandler) issueSession(user *models.User) (*dto.AuthData, error) {
	access, _, err := h.tokens.GenerateAccessToken(user)
	if err != nil {
		return nil, err
	}
	refresh, err := h.tokens.IssueRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}
	return &dto.AuthData{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "Bearer",
		ExpiresIn:    h.tokens.TTLSeconds(),
		User:         dto.NewUserProfile(user),
	}, nil
}
