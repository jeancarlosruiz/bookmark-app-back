package repositories

import (
	"time"

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

func (r *BookmarkRepository) FindByURL(url string, userID string) (*models.Bookmarks, error) {
	var bookmark models.Bookmarks
	err := r.db.Where("url = ? AND user_id = ?", url, userID).First(&bookmark).Error

	if err != nil {
		return nil, err
	}

	return &bookmark, nil
}

func (r *BookmarkRepository) Create(bookmark *models.Bookmarks) error {
	return r.db.Create(bookmark).Error
}

func (r *BookmarkRepository) FindByIDWithTags(id uint, userID string) (*models.Bookmarks, error) {
	var bookmark models.Bookmarks
	err := r.db.Preload("Tags").Where("id = ? AND user_id = ?", id, userID).First(&bookmark).Error

	if err != nil {
		return nil, err
	}

	return &bookmark, nil
}

func (r *BookmarkRepository) FindByUserIDWithTags(userID string) ([]models.Bookmarks, error) {
	var bookmarks []models.Bookmarks

	err := r.db.Preload("Tags").Where("user_id = ?", userID).Where("is_archived = ?", false).Order("pinned DESC, updated_at DESC").Find(&bookmarks).Error

	if err != nil {
		return nil, err
	}

	return bookmarks, nil
}

func (r *BookmarkRepository) FindByTags(tags []string, userID string) ([]models.Bookmarks, error) {

	var bookmarks []models.Bookmarks
	err := r.db.Model(&models.Bookmarks{}).Preload("Tags").Joins("JOIN bookmark_tags bt ON bt.bookmark_id = bookmarks.id").Joins("JOIN tags t ON t.id = bt.tag_id").Where("LOWER(t.title) IN ?", tags).Where("bookmarks.user_id = ?", userID).Find(&bookmarks).Error

	if err != nil {
		return nil, err
	}

	return bookmarks, nil
}

func (r *BookmarkRepository) FindBookmarkByTitle(title string, userID string) ([]models.Bookmarks, error) {

	var bookmarks []models.Bookmarks
	err := r.db.Where("user_id = ?", userID).Where("title ILIKE ?", "%"+title+"%").Preload("Tags").Find(&bookmarks).Error

	if err != nil {
		return nil, err
	}

	return bookmarks, nil

}

func (r *BookmarkRepository) SoftDeleteBookmarkByID(id uint, userID string) (*models.Bookmarks, error) {

	var bookmark models.Bookmarks

	err := r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&bookmark).Error

	if err != nil {
		return nil, err
	}

	return &bookmark, nil

}

func (r *BookmarkRepository) UpdateBookmarkByID(id uint, userID string, update map[string]interface{}) (*models.Bookmarks, error) {

	var bookmark models.Bookmarks

	err := r.db.Where("id = ? AND user_id = ?", id, userID).Save(update).Error

	if err != nil {

		return nil, err
	}

	return &bookmark, nil
}

func (r *BookmarkRepository) IncrementVisitCount(id uint, userID string) (*models.Bookmarks, error) {

	err := r.db.Model(&models.Bookmarks{}).Where("id = ? AND user_id = ?", id, userID).Updates(map[string]interface{}{
		"visit_count":  gorm.Expr("visit_count + 1"),
		"last_visited": time.Now(),
	}).Error

	if err != nil {

		return nil, err
	}

	bookmark, err := r.FindByIDWithTags(id, userID)

	if err != nil {
		return nil, err
	}

	return bookmark, nil
}

func (r *BookmarkRepository) TogglePinnedByID(id uint, userID string) (*models.Bookmarks, error) {

	err := r.db.Model(&models.Bookmarks{}).Where("id = ? AND user_id = ?", id, userID).Update("pinned", gorm.Expr("NOT pinned")).Error

	if err != nil {

		return nil, err
	}

	bookmark, err := r.FindByIDWithTags(id, userID)

	if err != nil {
		return nil, err
	}

	return bookmark, nil
}

func (r *BookmarkRepository) ToggleIsArchiveByID(id uint, userID string) (*models.Bookmarks, error) {

	err := r.db.Model(&models.Bookmarks{}).Where("id = ? AND user_id = ?", id, userID).Update("is_archived", gorm.Expr("NOT is_archived")).Error

	if err != nil {

		return nil, err
	}

	bookmark, err := r.FindByIDWithTags(id, userID)

	if err != nil {
		return nil, err
	}

	return bookmark, nil
}
