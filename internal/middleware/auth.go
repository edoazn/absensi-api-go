package middleware

import (
	"net/http"
	"strings"

	"github.com/edoazn/absensi-go/internal/api"
	"github.com/edoazn/absensi-go/internal/models"
	"github.com/edoazn/absensi-go/internal/services"
	"github.com/gin-gonic/gin"
)

const ctxClaimsKey = "auth.claims"

func JWTAuth(tokens *services.TokenService, blacklist *services.Blacklist) gin.HandlerFunc {
	const prefix = "Bearer "
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
			api.Fail(c, http.StatusUnauthorized, "Unauthenticated")
			return
		}
		claims, err := tokens.ParseAccessToken(header[len(prefix):])
		if err != nil || blacklist.IsBlacklisted(claims.ID) {
			api.Fail(c, http.StatusUnauthorized, "Unauthenticated")
			return
		}
		c.Set(ctxClaimsKey, claims)
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if claims := ClaimsFrom(c); claims == nil || claims.Role != models.RoleAdmin {
			api.Fail(c, http.StatusForbidden, "Forbidden")
			return
		}
		c.Next()
	}
}

func ClaimsFrom(c *gin.Context) *services.AccessTokenClaims {
	value, ok := c.Get(ctxClaimsKey)
	if !ok {
		return nil
	}
	claims, _ := value.(*services.AccessTokenClaims)
	return claims
}

func UserIDFrom(c *gin.Context) uint {
	if claims := ClaimsFrom(c); claims != nil {
		return claims.UserID
	}
	return 0
}
