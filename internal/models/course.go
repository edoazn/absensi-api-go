package models

import (
	"time"

	"gorm.io/gorm"
)

type Course struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	CourseName   string         `json:"course_name"`
	CourseCode   string         `json:"course_code"`
	LecturerName string         `json:"lecturer_name"`
	LocationRoom *string        `json:"location_room"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (Course) TableName() string { return "courses" }
