package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name   string
	Email  string `gorm:"unique"`
	NeonID string `gorm:"uniqueIndex"`
}
