package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name  string
	Email string `gorm:"unique"`
}

type Tag struct {
	gorm.Model
	Title string
}

type Bookmarks struct {
	gorm.Model
	Title       string `grom:"unique"`
	Url         string
	Favicon     string
	Description string
	Tags        []Tag
	Pinned      bool
	IsArchived  bool
	VisitCount  int
}
