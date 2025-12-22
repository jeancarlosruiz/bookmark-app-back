package models

import (
	"time"

	"gorm.io/gorm"
)

type Tag struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt"`
	Title     string         `gorm:"not null;index" json:"title"`
	UserID    string         `gorm:"index;not null" json:"userId"`
	Bookmarks []Bookmarks    `gorm:"many2many:bookmark_tags;foreignKey:ID;joinForeignKey:TagID;References:ID;joinReferences:BookmarkID" json:"-"`
}
