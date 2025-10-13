package models

import (
	"time"

	"gorm.io/gorm"
)

type Bookmarks struct {
	gorm.Model
	Title       string `gorm:"unique;not null"`
	Url         string `gorm:"unique;not null;index"`
	Favicon     string
	Description string
	Pinned      bool  `gorm:"default:false"`
	IsArchived  bool  `gorm:"default:false"`
	VisitCount  int   `gorm:"default:0"`
	Tags        []Tag `gorm:"many2many:bookmark_tags"`
  LastVisited time.Time
	timeUserID      uint
}
