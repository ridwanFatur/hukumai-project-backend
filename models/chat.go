package models

import (
	"time"

	"gorm.io/gorm"
)

// ChatSession represents a conversation thread belonging to a user
type ChatSession struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	Title     string         `gorm:"size:255" json:"title"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Chat represents a single message within a ChatSession.
// Role is either "user" or "assistant".
type Chat struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	ChatSessionID uint           `gorm:"not null;index" json:"chat_session_id"`
	Role          string         `gorm:"size:50;not null" json:"role"`
	Content       string         `gorm:"type:text;not null" json:"content"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}
