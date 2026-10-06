package models

import (
	"time"

	"gorm.io/gorm"
)

type Class struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Name         string         `json:"name"`
	AcademicYear string         `json:"academic_year"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Students  []User     `gorm:"many2many:class_user" json:"students,omitempty"`
	Schedules []Schedule `gorm:"foreignKey:ClassID" json:"-"`
}

func (Class) TableName() string { return "classes" }
