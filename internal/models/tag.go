package models

import "gorm.io/gorm"

type Tag struct {
	gorm.Model
	Title string `gorm:"unique;not null;index"`
}
