package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	RoleAdmin     = "admin"
	RoleMahasiswa = "mahasiswa"
)

type User struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	Name            string         `json:"name"`
	IdentityNumber  string         `json:"identity_number"`
	Email           *string        `json:"email"`
	EmailVerifiedAt *time.Time     `json:"email_verified_at"`
	Password        string         `json:"-"`
	Role            string         `gorm:"default:mahasiswa" json:"role"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Classes     []Class      `gorm:"many2many:class_user" json:"classes,omitempty"`
	Attendances []Attendance `gorm:"foreignKey:UserID" json:"-"`
}

func (User) TableName() string { return "users" }

func (u *User) IsAdmin() bool     { return u.Role == RoleAdmin }
func (u *User) IsMahasiswa() bool { return u.Role == RoleMahasiswa }
