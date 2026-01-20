package models

import (
	"time"

	"gorm.io/gorm"
)

type Bookmarks struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deletedAt"`
	Title       string         `gorm:"not null" json:"title"`
	Url         string         `gorm:"not null;uniqueIndex:idx_user_url,where:deleted_at IS NULL" json:"url"`
	Favicon     string         `json:"favicon"`
	Description string         `json:"description"`
	Pinned      bool           `gorm:"default:false" json:"pinned"`
	IsArchived  bool           `gorm:"default:false" json:"isArchived"`
	VisitCount  int            `gorm:"default:0" json:"visitCount"`
	Tags        []Tag          `gorm:"many2many:bookmark_tags;foreignKey:ID;joinForeignKey:BookmarkID;References:ID;joinReferences:TagID" json:"tags"`
	LastVisited *time.Time     `gorm:"default:null" json:"lastVisited"`

	// User relationship
	UserID string `gorm:"column:user_id;not null;uniqueIndex:idx_user_url,where:deleted_at IS NULL" json:"userId"`
	User   User   `gorm:"foreignKey:UserID;references:ID" json:"-"`
}
