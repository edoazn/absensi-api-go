package models

import (
	"time"
)

type AdminAuditLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `json:"user_id"`
	Action     string    `json:"action"` // mis. user.create, schedule.delete
	EntityType string    `json:"entity_type"`
	EntityID   *uint     `json:"entity_id"`
	Details    *string   `gorm:"type:json" json:"details"`
	IPAddress  string    `gorm:"column:ip_address" json:"ip_address"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (AdminAuditLog) TableName() string { return "admin_audit_logs" }
