package dto

import "github.com/edoazn/absensi-go/internal/models"

type LoginRequest struct {
	IdentityNumber string `json:"identity_number" binding:"required"`
	Password       string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type AuthData struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	TokenType    string      `json:"token_type"`
	ExpiresIn    int64       `json:"expires_in"`
	User         UserProfile `json:"user"`
}

type UserProfile struct {
	ID             uint    `json:"id"`
	Name           string  `json:"name"`
	IdentityNumber string  `json:"identity_number"`
	Email          *string `json:"email"`
	Role           string  `json:"role"`
}

func NewUserProfile(user *models.User) UserProfile {
	return UserProfile{
		ID:             user.ID,
		Name:           user.Name,
		IdentityNumber: user.IdentityNumber,
		Email:          user.Email,
		Role:           user.Role,
	}
}
