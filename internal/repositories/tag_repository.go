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

func (r *TagRepository) FindByUserID(userID string) ([]models.Tag, error) {
	var tags []models.Tag

	err := r.db.Where("user_id = ?", userID).Find(&tags).Error

	if err != nil {
		return nil, err
	}

	return tags, nil

}

func (r *TagRepository) FindByUserIDWithCount(userID string) ([]models.TagWithCount, error) {
	var tagsWithCount []models.TagWithCount

	err := r.db.Model(&models.Tag{}).
		Select("tags.*, COUNT(bookmarks.id) as total_bookmarks").
		Joins("LEFT JOIN bookmark_tags ON bookmark_tags.tag_id = tags.id").
		Joins("LEFT JOIN bookmarks ON bookmarks.id = bookmark_tags.bookmark_id AND bookmarks.deleted_at IS NULL").
		Where("tags.user_id = ?", userID).
		Group("tags.id").
		Order("tags.title ASC").
		Scan(&tagsWithCount).Error

	if err != nil {
		return nil, err
	}

	return tagsWithCount, nil
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

func (r *TagRepository) FindByID(id string) (*models.Tag, error) {
	var tag models.Tag

	err := r.db.Where("id = ?", id).First(&tag).Error

	if err != nil {
		return nil, err
	}

	return &tag, nil
}

func (r *TagRepository) Update(tag *models.Tag) error {
	return r.db.Save(tag).Error
}
