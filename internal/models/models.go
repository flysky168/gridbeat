package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Model a basic GoLang struct which includes the following fields: ID, CreatedAt, UpdatedAt, DeletedAt
// It may be embedded into your model or you may build your own model without it
//
//	type User struct {
//	  Base
//	}
type Base struct {
	ID        string `gorm:"primaryKey;type:char(36)" json:"id"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (b *Base) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.NewString() // 形如 "550e8400-e29b-41d4-a716-446655440000"
	}
	return nil
}
