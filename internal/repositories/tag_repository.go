package repositories

import (
	"github.com/jeancarlosruiz/bookmark-app-back/internal/database"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/models"
	"gorm.io/gorm"
)

type TagRepository struct {
	db *gorm.DB
}

func NewTagRepository() *TagRepository {
	return &TagRepository{
		db: database.DB,
	}
}

func (r *TagRepository) FindByTitleAndUserID(title string, userID string) (*models.Tag, error) {

	var tag models.Tag

	err := r.db.Where(&models.Tag{Title: title, UserID: userID}).First(&tag).Error

	if err != nil {
		return nil, err
	}

	return &tag, nil
}

func (r *TagRepository) Create(tag *models.Tag) error {
	return r.db.Create(tag).Error
}
