package models

import "time"

type BookmarkTag struct {
	BookmarkID uint `gorm:"primaryKey"`
	TagID      uint `gorm:"primaryKey"`
	CreatedAt  time.Time
}
