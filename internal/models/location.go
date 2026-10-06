package models

import (
	"time"

	"gorm.io/gorm"
)

type Location struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `json:"name"`
	Latitude  float64        `gorm:"column:latitude" json:"latitude"`
	Longitude float64        `gorm:"column:longitude" json:"longitude"`
	Radius    float64        `json:"radius"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (Location) TableName() string { return "locations" }
