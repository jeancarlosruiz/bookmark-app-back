package models

import (
	"time"
)

type User struct {
	ID        string     `gorm:"column:id;primaryKey" json:"id"`
	Name      string     `gorm:"column:name" json:"name"`
	Email     string     `gorm:"column:email" json:"email"`
	CreatedAt time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt *time.Time `gorm:"column:updated_at" json:"updated_at,omitempty"`
	DeletedAt *time.Time `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
	RawJSON   []byte     `gorm:"column:raw_json;type:jsonb" json:"raw_json,omitempty"`
}

func (User) TableName() string {
	return "neon_auth.users_sync"
}
