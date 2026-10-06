package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/edoazn/absensi-go/internal/api"
	"github.com/edoazn/absensi-go/internal/middleware"
	"github.com/edoazn/absensi-go/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// recordAudit mencatat aksi mutasi admin secara best-effort: kegagalan
// pencatatan tidak boleh membatalkan request bisnis yang sudah sukses.
// Field "password" selalu dibuang dari details.
func recordAudit(db *gorm.DB, c *gin.Context, action, entityType string, entityID *uint, details map[string]any) {
	payload := make(map[string]any, len(details)+1)
	for k, v := range details {
		if k == "password" {
			continue
		}
		payload[k] = v
	}
	var encoded *string
	if raw, err := json.Marshal(payload); err == nil {
		s := string(raw)
		encoded = &s
	}

	entry := models.AdminAuditLog{
		UserID:     middleware.UserIDFrom(c),
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		Details:    encoded,
		IPAddress:  c.ClientIP(),
	}
	_ = db.Create(&entry).Error
}

type AuditHandler struct {
	db *gorm.DB
	tz *time.Location
}

func NewAuditHandler(db *gorm.DB, tz *time.Location) *AuditHandler {
	return &AuditHandler{db: db, tz: tz}
}

func (h *AuditHandler) Index(c *gin.Context) {
	page := queryIntOr(c, "page", 1)
	perPage := queryIntOr(c, "per_page", 15)
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 15
	}

	query := h.db.Model(&models.AdminAuditLog{})
	if action := c.Query("action"); action != "" {
		query = query.Where("action = ?", action)
	}
	if uid := queryIntOr(c, "user_id", 0); uid > 0 {
		query = query.Where("user_id = ?", uid)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal memuat audit log")
		return
	}

	var rows []models.AdminAuditLog
	if err := query.
		Preload("User").
		Order("created_at DESC, id DESC").
		Limit(perPage).
		Offset((page - 1) * perPage).
		Find(&rows).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal memuat audit log")
		return
	}

	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		var details json.RawMessage
		if row.Details != nil {
			details = json.RawMessage(*row.Details)
		}
		items = append(items, gin.H{
			"id":          row.ID,
			"user_id":     row.UserID,
			"user":        row.User.Name,
			"action":      row.Action,
			"entity_type": row.EntityType,
			"entity_id":   row.EntityID,
			"details":     details,
			"ip_address":  row.IPAddress,
			"created_at":  row.CreatedAt.In(h.tz).Format("2006-01-02 15:04:05"),
		})
	}

	lastPage := int(total) / perPage
	if int(total)%perPage > 0 {
		lastPage++
	}
	if lastPage < 1 {
		lastPage = 1
	}

	api.OK(c, gin.H{
		"items": items,
		"meta": gin.H{
			"current_page": page,
			"per_page":     perPage,
			"total":        total,
			"last_page":    lastPage,
		},
	}, "Audit log")
}
