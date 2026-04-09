package models

import (
	"time"

	"gorm.io/gorm"
)

type SubscriptionPlan struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Name          string         `gorm:"size:50;not null" json:"name"`
	Slug          string         `gorm:"size:50;uniqueIndex;not null" json:"slug"`
	Price         int64          `json:"price"` // price in IDR, 0 for free
	Currency      string         `gorm:"size:10;default:'IDR'" json:"currency"`
	Interval      string         `gorm:"size:20;default:'month'" json:"interval"`
	Features      string         `gorm:"type:text" json:"features"` // JSON array of strings
	StripePriceID string         `gorm:"size:255" json:"stripe_price_id,omitempty"`
	IsActive      bool           `gorm:"default:true" json:"is_active"`
	SortOrder     int            `json:"sort_order"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

type Subscription struct {
	ID                   uint             `gorm:"primaryKey" json:"id"`
	UserID               uint             `gorm:"not null;index" json:"user_id"`
	PlanID               uint             `gorm:"not null" json:"plan_id"`
	Plan                 SubscriptionPlan `gorm:"foreignKey:PlanID" json:"plan"`
	Status               string           `gorm:"size:50;not null;default:'active'" json:"status"` // active, canceled, past_due
	StripeSubscriptionID string           `gorm:"size:255" json:"stripe_subscription_id,omitempty"`
	StripeCustomerID     string           `gorm:"size:255" json:"stripe_customer_id,omitempty"`
	CurrentPeriodStart   *time.Time       `json:"current_period_start,omitempty"`
	CurrentPeriodEnd     *time.Time       `json:"current_period_end,omitempty"`
	CreatedAt            time.Time        `json:"created_at"`
	UpdatedAt            time.Time        `json:"updated_at"`
	DeletedAt            gorm.DeletedAt   `gorm:"index" json:"-"`
}
