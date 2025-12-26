package services

import (
	"context"
	"errors"

	"github.com/jeancarlosruiz/bookmark-app-back/internal/models"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/repositories"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/validator"
	"gorm.io/gorm"
)

type BookmarkService struct {
	bookmarkRepo *repositories.BookmarkRepository
	tagService   *TagService
	cacheService *CacheService
}

func NewBookmarkService() *BookmarkService {
	return &BookmarkService{
		bookmarkRepo: repositories.NewBookmarkRepository(),
		tagService:   NewTagService(),
		cacheService: &CacheService{},
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

	bookmark, err = s.bookmarkRepo.FindByIDWithTags(bookmark.ID, bookmark.UserID)

	if err != nil {
		return nil, err
	}

	// INVALIDACIÓN DE CACHÉ: Crucial después de crear un bookmark
	// El caché del usuario ahora está desactualizado
	ctx := context.Background()
	_ = s.cacheService.InvalidateUserCache(ctx, data.UserID)

	return bookmark, nil
}

func (s *BookmarkService) FindByIDWithTagsService(id uint, userID string) (*models.Bookmarks, error) {

	bookmark, err := s.bookmarkRepo.FindByIDWithTags(id, userID)

	if err != nil {
		return nil, err
	}

	return bookmark, nil

}

func (s *BookmarkService) FindByUserIDWithTagService(userID string) ([]models.Bookmarks, error) {
	ctx := context.Background()

	// CACHE HIT PATH: Intentar obtener desde caché
	cachedBookmarks, hit, err := s.cacheService.GetBookmarksFromCache(ctx, userID)
	if hit && err == nil {
		return cachedBookmarks, nil
	}

	// CACHE MISS PATH: Consultar base de datos
	bookmarks, err := s.bookmarkRepo.FindByUserIDWithTags(userID)

	if err == gorm.ErrRecordNotFound {
		return nil, ErrBookmarksNotFound
	}

	if err != nil {
		return nil, err
	}

	// Guardar en caché para futuras consultas
	// No retornamos error si el caché falla - la app continúa funcionando
	_ = s.cacheService.SetBookmarksCache(ctx, userID, bookmarks)

	return bookmarks, nil

}

func (s *BookmarkService) FindArchivedByUserIDWithTagService(userID string) ([]models.Bookmarks, error) {
	ctx := context.Background()

	// CACHE HIT PATH: Intentar obtener desde caché de ARCHIVADOS
	cachedBookmarks, hit, err := s.cacheService.GetArchivedBookmarksFromCache(ctx, userID)
	if hit && err == nil {
		return cachedBookmarks, nil
	}

	// CACHE MISS PATH: Consultar base de datos
	bookmarks, err := s.bookmarkRepo.FindArchivedByUserIDWithTags(userID)

	if err == gorm.ErrRecordNotFound {
		return nil, ErrBookmarksNotFound
	}

	if err != nil {
		return nil, err
	}

	// Guardar en caché de ARCHIVADOS para futuras consultas
	// No retornamos error si el caché falla - la app continúa funcionando
	_ = s.cacheService.SetArchivedBookmarksCache(ctx, userID, bookmarks)

	return bookmarks, nil

}

func (s *BookmarkService) FindBookmarksByTagsService(tags []string, userID string) ([]models.Bookmarks, error) {

	bookmarks, err := s.bookmarkRepo.FindByTags(tags, userID)

	if err == gorm.ErrRecordNotFound {
		return nil, ErrBookmarksNotFound
	}

	if err != nil {
		return nil, err
	}

	return bookmarks, nil
}

func (s *BookmarkService) FindBookmarkByTitleService(title string, userID string) ([]models.Bookmarks, error) {

	bookmarks, err := s.bookmarkRepo.FindBookmarkByTitle(title, userID)

	if err == gorm.ErrRecordNotFound {
		return nil, ErrBookmarksNotFound
	}

	if err != nil {
		return nil, err
	}

	return bookmarks, nil
}

func (s *BookmarkService) SoftDeleteBookmarkByIDService(id uint, userID string) (*models.Bookmarks, error) {
	bookmark, err := s.bookmarkRepo.SoftDeleteBookmarkByID(id, userID)

	if err != nil {
		return nil, err
	}

	// INVALIDACIÓN DE CACHÉ: Crucial después de eliminar un bookmark
	ctx := context.Background()
	_ = s.cacheService.InvalidateUserCache(ctx, userID)

	return bookmark, nil
}

func (s *BookmarkService) IncrementVisitCountService(id uint, userID string) (*models.Bookmarks, error) {
	bookmark, err := s.bookmarkRepo.IncrementVisitCount(id, userID)

	if err != nil {
		return nil, err
	}

	return bookmark, nil
}

func (s *BookmarkService) TogglePinnedByIDService(id uint, userID string) (*models.Bookmarks, error) {
	bookmark, err := s.bookmarkRepo.TogglePinnedByID(id, userID)

	if err != nil {
		return nil, err
	}

	// INVALIDACIÓN DE CACHÉ: Crucial después de cambiar el estado de un bookmark
	ctx := context.Background()
	_ = s.cacheService.InvalidateUserCache(ctx, userID)

	return bookmark, nil
}

func (s *BookmarkService) ToggleIsArchiveByIDService(id uint, userID string) (*models.Bookmarks, error) {
	bookmark, err := s.bookmarkRepo.ToggleIsArchiveByID(id, userID)

	if err != nil {
		return nil, err
	}

	// INVALIDACIÓN DE CACHÉ: Crucial después de cambiar el estado de un bookmark
	ctx := context.Background()
	_ = s.cacheService.InvalidateUserCache(ctx, userID)

	return bookmark, nil
}

func (s *BookmarkService) UpdateBookmarkService(id uint, userID string, data validator.UpdateBookmark) (*models.Bookmarks, error) {
	updates := make(map[string]interface{})

	if data.Title != nil {
		updates["title"] = *data.Title
	}

	if data.Url != nil {
		updates["url"] = *data.Url
	}

	if data.Description != nil {
		updates["description"] = *data.Description
	}
	if data.Favicon != nil {
		updates["favicon"] = *data.Favicon
	}
	if data.Pinned != nil {
		updates["pinned"] = *data.Pinned
	}
	if data.IsArchived != nil {
		updates["is_archived"] = *data.IsArchived
	}

	if data.Tags != nil {
		_, err := s.tagService.FindOrCreateTags(data.Tags, userID)

		if err != nil {
			return nil, err
		}

		// Hacer el replacement de tags

	}

	if len(updates) > 0 {
		_, err := s.bookmarkRepo.UpdateBookmarkByID(id, userID, updates)

		if err != nil {
			return nil, err
		}

	}

	bookmark, err := s.bookmarkRepo.FindByIDWithTags(id, userID)
	if err != nil {
		return nil, err
	}

	// INVALIDACIÓN DE CACHÉ: Crucial después de actualizar un bookmark
	ctx := context.Background()
	_ = s.cacheService.InvalidateUserCache(ctx, userID)

	return bookmark, nil
}

func (s *BookmarkService) CheckURLExists(url string, userID string) error {
	_, err := s.bookmarkRepo.FindByURL(url, userID)

	if err == nil {
		// El bookmark existe
		return ErrBookmarkURLAlreadyExists
	}

	if err == gorm.ErrRecordNotFound {
		// No existe, todo bien
		return nil
	}

	// Otro tipo de error de base de datos
	return err
}

var (
	ErrBookmarkAlreadyExists    = errors.New("bookmark with this title or URL already exists")
	ErrBookmarksNotFound        = errors.New("Bookmarks not found")
	ErrBookmarkURLAlreadyExists = errors.New("bookmark with this URL already exists for this user")
)
