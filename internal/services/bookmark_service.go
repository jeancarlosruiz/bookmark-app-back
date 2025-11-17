package services

import (
	"errors"

	"github.com/jeancarlosruiz/bookmark-app-back/internal/models"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/repositories"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/validator"
	"gorm.io/gorm"
)

type BookmarkService struct {
	bookmarkRepo *repositories.BookmarkRepository
	tagService   *TagService
}

func NewBookmarkService() *BookmarkService {
	return &BookmarkService{
		bookmarkRepo: repositories.NewBookmarkRepository(),
		tagService:   NewTagService(),
	}
}

func (s *BookmarkService) CreateBookmarkService(data validator.CreateBookmark) (*models.Bookmarks, error) {
	existing, _ := s.bookmarkRepo.FindByTitleOrURL(data.Title, data.Url, data.UserID)

	if existing != nil {
		return nil, ErrBookmarkAlreadyExists
	}

	tags, err := s.tagService.FindOrCreateTags(data.Tags, data.UserID)

	if err != nil {
		return nil, err
	}

	bookmark := &models.Bookmarks{
		Title:  data.Title,
		Url:    data.Url,
		UserID: data.UserID,
		Tags:   tags,
	}

	if err := s.bookmarkRepo.Create(bookmark); err != nil {
		return nil, err
	}

	bookmark, err = s.bookmarkRepo.FindByIDWithTags(bookmark.ID)

	if err != nil {
		return nil, err
	}

	return bookmark, nil
}

func (s *BookmarkService) FindByIDWithTagsService(id uint) (*models.Bookmarks, error) {

	bookmark, err := s.bookmarkRepo.FindByIDWithTags(id)

	if err != nil {
		return nil, err
	}

	return bookmark, nil

}

func (s *BookmarkService) FindByUserIDWithTagService(userID string) ([]models.Bookmarks, error) {

	bookmarks, err := s.bookmarkRepo.FindByUserIDWithTags(userID)

	if err == gorm.ErrRecordNotFound {
		return nil, ErrBookmarksNotFound
	}

	if err != nil {
		return nil, err
	}

	return bookmarks, nil

}

func (s *BookmarkService) FindBookmarksByTagsService(tags []string) ([]models.Bookmarks, error) {

	bookmarks, err := s.bookmarkRepo.FindByTags(tags)

	if err == gorm.ErrRecordNotFound {
		return nil, ErrBookmarksNotFound
	}

	if err != nil {
		return nil, err
	}

	return bookmarks, nil
}

func (s *BookmarkService) FindBookmarkByTitleService(title string) ([]models.Bookmarks, error) {

	bookmarks, err := s.bookmarkRepo.FindBookmarkByTitle(title)

	if err == gorm.ErrRecordNotFound {
		return nil, ErrBookmarksNotFound
	}

	if err != nil {
		return nil, err
	}

	return bookmarks, nil
}

func (s *BookmarkService) SoftDeleteBookmarkByIDService(id uint) (*models.Bookmarks, error) {
	bookmark, err := s.bookmarkRepo.SoftDeleteBookmarkByID(id)

	if err != nil {
		return nil, err
	}

	return bookmark, nil
}

var (
	ErrBookmarkAlreadyExists = errors.New("bookmark with this title or URL already exists")
	ErrBookmarksNotFound     = errors.New("Bookmarks not found")
)
