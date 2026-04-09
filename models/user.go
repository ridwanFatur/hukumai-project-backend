package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID        uint           `gorm:"primaryKey"`           // Auto increment primary key
	Email     string         `gorm:"uniqueIndex;not null"` // Unique & required
	Name      string         `gorm:"size:100"`             // Optional name
	CreatedAt time.Time      // Auto-set by GORM
	UpdatedAt time.Time      // Auto-set by GORM
	DeletedAt gorm.DeletedAt `gorm:"index"` // Soft delete
}
