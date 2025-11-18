package repositories

import (
	"github.com/jeancarlosruiz/bookmark-app-back/internal/database"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/models"
	"gorm.io/gorm"
)

type BookmarkRepository struct {
	db *gorm.DB
}

func NewBookmarkRepository() *BookmarkRepository {
	return &BookmarkRepository{
		db: database.DB,
	}
}

func (r *BookmarkRepository) FindByTitleOrURL(title string, url string, userID string) (*models.Bookmarks, error) {
	var bookmark models.Bookmarks
	err := r.db.Where(&models.Bookmarks{
		Title:  title,
		Url:    url,
		UserID: userID,
	}).First(&bookmark).Error

	if err != nil {
		return nil, err
	}

	return &bookmark, nil

}

func (r *BookmarkRepository) Create(bookmark *models.Bookmarks) error {
	return r.db.Create(bookmark).Error
}

func (r *BookmarkRepository) FindByIDWithTags(id uint) (*models.Bookmarks, error) {
	var bookmark models.Bookmarks
	err := r.db.Preload("Tags").First(&bookmark, id).Error

	if err != nil {
		return nil, err
	}

	return &bookmark, nil
}

func (r *BookmarkRepository) FindByUserIDWithTags(userID string) ([]models.Bookmarks, error) {
	var bookmarks []models.Bookmarks

	err := r.db.Preload("Tags").Where("user_id = ?", userID).Find(&bookmarks).Error

	if err != nil {
		return nil, err
	}

	return bookmarks, nil
}

func (r *BookmarkRepository) FindByTags(tags []string) ([]models.Bookmarks, error) {

	var bookmarks []models.Bookmarks
	err := r.db.Model(&models.Bookmarks{}).Preload("Tags").Joins("JOIN bookmark_tags bt ON bt.bookmark_id = bookmarks.id").Joins("JOIN tags t ON t.id = bt.tag_id").Where("LOWER(t.title) IN ?", tags).Find(&bookmarks).Error

	if err != nil {
		return nil, err
	}

	return bookmarks, nil
}

func (r *BookmarkRepository) FindBookmarkByTitle(title string) ([]models.Bookmarks, error) {

	var bookmarks []models.Bookmarks
	err := r.db.Where("title ILIKE ?", "%"+title+"%").Preload("Tags").Find(&bookmarks).Error

	if err != nil {
		return nil, err
	}

	return bookmarks, nil

}

func (r *BookmarkRepository) SoftDeleteBookmarkByID(id uint) (*models.Bookmarks, error) {

	var bookmark models.Bookmarks

	err := r.db.Where("id = ?", id).Delete(&bookmark).Error

	if err != nil {
		return nil, err
	}

	return &bookmark, nil

}

func (r *BookmarkRepository) UpdateBookmarkByID(id uint, update map[string]interface{}) (*models.Bookmarks, error) {

	var bookmark models.Bookmarks

	err := r.db.Where("id = ?", id).Save(update).Error

	if err != nil {

		return nil, err
	}

	return &bookmark, nil
}
