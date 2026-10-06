package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	StatusHadir     = "hadir"
	StatusDitolak   = "ditolak"
	StatusTerlambat = "terlambat"
)

const (
	MethodGeolocation    = "geolocation"
	MethodQrCode         = "qr_code"
	MethodAttendanceCode = "attendance_code"
)

type Attendance struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	UserID      uint           `json:"user_id"`
	ScheduleID  uint           `json:"schedule_id"`
	Latitude    *float64       `json:"latitude"`
	Longitude   *float64       `json:"longitude"`
	Distance    *float64       `json:"distance"`
	GpsAccuracy *float64       `gorm:"column:gps_accuracy" json:"gps_accuracy"`
	Status      string         `json:"status"`
	Method      string         `json:"method"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	User     User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Schedule Schedule `gorm:"foreignKey:ScheduleID" json:"-"`
}

func (Attendance) TableName() string { return "attendances" }
