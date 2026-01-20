package services

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/jeancarlosruiz/bookmark-app-back/internal/models"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/repositories"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/utils"
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
	// Normalize the URL to ensure consistent storage and comparison
	normalizedURL, err := utils.NormalizeURL(data.Url)
	if err != nil {
		return nil, err
	}

	existing, err := s.bookmarkRepo.FindByTitleOrURL(data.Title, normalizedURL, data.UserID)

	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if existing != nil {
		return nil, ErrBookmarkAlreadyExists
	}

	tags, err := s.tagService.FindOrCreateTags(data.Tags, data.UserID)

	if err != nil {
		return nil, err
	}

	bookmark := &models.Bookmarks{
		Title:       data.Title,
		Url:         normalizedURL,
		Description: data.Description,
		UserID:      data.UserID,
		Favicon:     data.Favicon,
		Tags:        tags,
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

func (s *BookmarkService) FindByUserIDWithTagService(userID string, sortParam string, pagination *models.PaginationParams) ([]models.Bookmarks, models.PaginationMetadata, error) {
	ctx := context.Background()

	// CACHE HIT PATH: Intentar obtener desde caché
	cachedBookmarks, hit, err := s.cacheService.GetBookmarksFromCache(ctx, userID)
	if hit && err == nil {
		// Ordenar en memoria según sortParam
		sortBookmarks(cachedBookmarks, sortParam)

		if pagination != nil {
			paginatedBookmarks, metadata := PaginateBookmarks(cachedBookmarks, *pagination)

			return paginatedBookmarks, metadata, nil
		}

		emptyMetadata := models.PaginationMetadata{}

		return cachedBookmarks, emptyMetadata, nil
	}

	// CACHE MISS PATH: Consultar base de datos
	bookmarks, err := s.bookmarkRepo.FindByUserIDWithTags(userID)

	if err == gorm.ErrRecordNotFound {
		emptyMetadata := models.PaginationMetadata{}
		return nil, emptyMetadata, ErrBookmarksNotFound
	}

	if err != nil {

		emptyMetadata := models.PaginationMetadata{}
		return nil, emptyMetadata, err
	}

	// Guardar en caché SIN ordenar (para reutilizar con diferentes sorts)
	// No retornamos error si el caché falla - la app continúa funcionando
	_ = s.cacheService.SetBookmarksCache(ctx, userID, bookmarks)

	// Ordenar en memoria según sortParam antes de retornar
	sortBookmarks(bookmarks, sortParam)

	if pagination != nil {
		paginatedBookmarks, metadata := PaginateBookmarks(bookmarks, *pagination)

		return paginatedBookmarks, metadata, nil
	}

	emptyMetadata := models.PaginationMetadata{}
	return bookmarks, emptyMetadata, nil

}

func (s *BookmarkService) FindArchivedByUserIDWithTagService(userID string, sortParam string, pagination *models.PaginationParams) ([]models.Bookmarks, models.PaginationMetadata, error) {
	ctx := context.Background()

	// CACHE HIT PATH: Intentar obtener desde caché de ARCHIVADOS
	cachedBookmarks, hit, err := s.cacheService.GetArchivedBookmarksFromCache(ctx, userID)
	if hit && err == nil {
		// Ordenar en memoria según sortParam
		sortBookmarks(cachedBookmarks, sortParam)

		if pagination != nil {
			paginatedBookmarks, metadata := PaginateBookmarks(cachedBookmarks, *pagination)

			return paginatedBookmarks, metadata, nil
		}

		emptyMetadata := models.PaginationMetadata{}

		return cachedBookmarks, emptyMetadata, nil
	}

	// CACHE MISS PATH: Consultar base de datos
	bookmarks, err := s.bookmarkRepo.FindArchivedByUserIDWithTags(userID)

	if err == gorm.ErrRecordNotFound {

		emptyMetadata := models.PaginationMetadata{}
		return nil, emptyMetadata, ErrBookmarksNotFound
	}

	if err != nil {

		emptyMetadata := models.PaginationMetadata{}
		return nil, emptyMetadata, err
	}

	// Guardar en caché de ARCHIVADOS SIN ordenar (para reutilizar con diferentes sorts)
	// No retornamos error si el caché falla - la app continúa funcionando
	_ = s.cacheService.SetArchivedBookmarksCache(ctx, userID, bookmarks)

	// Ordenar en memoria según sortParam antes de retornar
	sortBookmarks(bookmarks, sortParam)

	if pagination != nil {
		paginatedBookmarks, metadata := PaginateBookmarks(bookmarks, *pagination)

		return paginatedBookmarks, metadata, nil
	}

	emptyMetadata := models.PaginationMetadata{}
	return bookmarks, emptyMetadata, nil

}

func (s *BookmarkService) FindBookmarksByTagsService(tags string, userID string, sortParam string, pagination *models.PaginationParams) ([]models.Bookmarks, models.PaginationMetadata, error) {

	var tagList = strings.Split(tags, ",")

	var searchList []string

	for _, tag := range tagList {

		tag = strings.TrimSpace(strings.ToLower(tag))

		if tag == "" {
			continue
		}

		searchList = append(searchList, tag)

	}

	emptyMetadata := models.PaginationMetadata{}
	bookmarks, err := s.bookmarkRepo.FindByTags(searchList, userID)

	if err == gorm.ErrRecordNotFound {
		return nil, emptyMetadata, ErrBookmarksNotFound
	}

	if err != nil {
		return nil, emptyMetadata, err
	}

	// Ordenar en memoria según sortParam antes de retornar
	// Las búsquedas por tags NO se cachean (queries dinámicas)
	sortBookmarks(bookmarks, sortParam)

	if pagination != nil {
		paginatedBookmarks, metadata := PaginateBookmarks(bookmarks, *pagination)

		return paginatedBookmarks, metadata, nil
	}

	return bookmarks, emptyMetadata, nil
}

func (s *BookmarkService) FindBookmarkByTitleService(title string, userID string, sortParam string, pagination *models.PaginationParams) ([]models.Bookmarks, models.PaginationMetadata, error) {
	emptyMetadata := models.PaginationMetadata{}

	bookmarks, err := s.bookmarkRepo.FindBookmarkByTitle(title, userID)

	if err == gorm.ErrRecordNotFound {
		return nil, emptyMetadata, ErrBookmarksNotFound
	}

	if err != nil {
		return nil, emptyMetadata, err
	}

	sortBookmarks(bookmarks, sortParam)

	if pagination != nil {
		paginatedBookmarks, metadata := PaginateBookmarks(bookmarks, *pagination)
		return paginatedBookmarks, metadata, nil
	}

	return bookmarks, emptyMetadata, nil
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

	// INVALIDACIÓN DE CACHÉ: Crucial después de cambiar el estado de un bookmark
	ctx := context.Background()
	_ = s.cacheService.InvalidateUserCache(ctx, userID)

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
		// Normalize the URL to ensure consistent storage and comparison
		normalizedURL, err := utils.NormalizeURL(*data.Url)
		if err != nil {
			return nil, err
		}
		updates["url"] = normalizedURL
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
		err := s.UpdateBookmarkTags(id, userID, data.Tags)

		if err != nil {
			return nil, err
		}

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
	// Normalize the URL to ensure consistent comparison
	normalizedURL, err := utils.NormalizeURL(url)
	if err != nil {
		return err
	}

	_, err = s.bookmarkRepo.FindByURL(normalizedURL, userID)

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

// sortBookmarks ordena un slice de bookmarks en memoria con prioridad a pinned
// PRIORIDAD: Los bookmarks con pinned=true SIEMPRE van primero
// Valores de sortParam aceptados: "created" (más reciente primero), "visited" (último visitado primero), "count" (más visitado primero)
// Si sortParam está vacío o es inválido, se ordena por CreatedAt descendente (por defecto)
func sortBookmarks(bookmarks []models.Bookmarks, sortParam string) {
	sort.Slice(bookmarks, func(i, j int) bool {
		// PRIORIDAD 1: Pinned siempre va primero
		if bookmarks[i].Pinned != bookmarks[j].Pinned {
			return bookmarks[i].Pinned // true > false
		}

		// PRIORIDAD 2: Aplicar ordenamiento según sortParam (dentro del mismo grupo de pinned)
		switch sortParam {
		case "created":
			// Ordenar por CreatedAt descendente (más reciente primero)
			return bookmarks[i].CreatedAt.After(bookmarks[j].CreatedAt)

		case "visited":
			// Ordenar por LastVisited descendente (último visitado primero)
			// Los bookmarks sin visitar (LastVisited == nil) van al final
			if bookmarks[i].LastVisited != nil && bookmarks[j].LastVisited != nil {
				return bookmarks[i].LastVisited.After(*bookmarks[j].LastVisited)
			}
			// Si solo i tiene LastVisited, va primero
			if bookmarks[i].LastVisited != nil {
				return true
			}
			// Si solo j tiene LastVisited, va primero
			if bookmarks[j].LastVisited != nil {
				return false
			}
			// Si ninguno tiene LastVisited, ordenar por CreatedAt
			return bookmarks[i].CreatedAt.After(bookmarks[j].CreatedAt)

		case "count":
			// Ordenar por VisitCount descendente (más visitado primero)
			if bookmarks[i].VisitCount == bookmarks[j].VisitCount {
				return bookmarks[i].CreatedAt.After(bookmarks[j].CreatedAt)
			}
			return bookmarks[i].VisitCount > bookmarks[j].VisitCount

		default:
			// Por defecto: ordenar por CreatedAt descendente
			return bookmarks[i].CreatedAt.After(bookmarks[j].CreatedAt)
		}
	})
}

// filterByTitle filtra bookmarks por título usando búsqueda case-insensitive
// Implementa la misma lógica que el ILIKE '%title%' de PostgreSQL
func filterByTitle(bookmarks []models.Bookmarks, title string) []models.Bookmarks {
	// Si no hay término de búsqueda, retornar todos
	if title == "" {
		return bookmarks
	}

	// Convertir el término de búsqueda a minúsculas
	searchTerm := strings.ToLower(title)
	filtered := make([]models.Bookmarks, 0)

	// Filtrar bookmarks que contengan el término en el título
	for _, bookmark := range bookmarks {
		if strings.Contains(strings.ToLower(bookmark.Title), searchTerm) {
			filtered = append(filtered, bookmark)
		}
	}

	return filtered
}

func (s *BookmarkService) UpdateBookmarkTags(bookmarkID uint, userID string, tagNames []string) error {
	bookmark, err := s.bookmarkRepo.FindByIDWithTags(bookmarkID, userID)

	if err != nil {
		return err
	}

	tagIDs, err := s.bookmarkRepo.GetTagsId(bookmark.ID)

	if err != nil {
		return err
	}

	tags, err := s.tagService.FindOrCreateTags(tagNames, userID)

	if err != nil {
		return err
	}

	var currentTagIDs []uint

	for _, tag := range tags {
		currentTagIDs = append(currentTagIDs, tag.ID)
	}

	toRemove, toAdd := calculateTagDifferences(tagIDs, currentTagIDs)

	if len(toRemove) > 0 {

		err := s.bookmarkRepo.RemoveTagAssociations(bookmarkID, toRemove)

		if err != nil {
			return err
		}
	}

	if len(toAdd) > 0 {
		err := s.bookmarkRepo.AddTagAssociations(bookmarkID, toAdd)

		if err != nil {
			return err
		}
	}

	ctx := context.Background()

	_ = s.cacheService.InvalidateUserCache(ctx, userID)

	return nil

}

func PaginateBookmarks(bookmarks []models.Bookmarks, params models.PaginationParams) ([]models.Bookmarks, models.PaginationMetadata) {
	total := len(bookmarks)

	metadata := params.CalculateMetadata(total)

	if total == 0 {
		return []models.Bookmarks{}, metadata
	}

	offset := params.GetOffset()
	end := params.GetEndIndex(total)

	if offset >= total {
		return []models.Bookmarks{}, metadata
	}

	paginatedBookmarks := bookmarks[offset:end]

	return paginatedBookmarks, metadata
}

func calculateTagDifferences(currentIDs, newIDs []uint) (toRemove, toAdd []uint) {
	currentSet := make(map[uint]struct{}, len(currentIDs))

	for _, id := range currentIDs {
		currentSet[id] = struct{}{}
	}

	newSet := make(map[uint]struct{}, len(newIDs))

	for _, id := range newIDs {
		newSet[id] = struct{}{}
	}

	for _, id := range currentIDs {
		if _, exists := newSet[id]; !exists {
			toRemove = append(toRemove, id)
		}
	}

	for _, id := range newIDs {
		if _, exists := currentSet[id]; !exists {
			toAdd = append(toAdd, id)
		}
	}

	return
}

var (
	ErrBookmarkAlreadyExists    = errors.New("bookmark already exists")
	ErrBookmarksNotFound        = errors.New("bookmarks not found")
	ErrBookmarkURLAlreadyExists = errors.New("URL already exists")
)
