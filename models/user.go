package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	ClerkID   string         `gorm:"uniqueIndex;size:255" json:"clerk_id"`
	Email     string         `gorm:"uniqueIndex;not null" json:"email"`
	Name      string         `gorm:"size:100" json:"name"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
