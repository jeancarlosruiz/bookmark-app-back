package models

import "gorm.io/gorm"

type Tag struct {
	gorm.Model
	Title     string      `gorm:"unique;not null;index"`
	Bookmarks []Bookmarks `gorm:"many2many:bookmark_tags;foreignKey:ID;joinForeignKey:TagID;References:ID;joinReferences:BookmarkID" json:"-"`
}
