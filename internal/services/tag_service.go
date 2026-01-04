package services

import (
	"context"
	"errors"
	"strings"

	"github.com/jeancarlosruiz/bookmark-app-back/internal/models"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/repositories"
	"gorm.io/gorm"
)

var (
	ErrTagAlreadyExists = errors.New("tag already exists")
	ErrTagNotFound      = errors.New("tag not found")
	ErrTagUnauthorized  = errors.New("unauthorized to modify tag")
	ErrTagHasBookmarks  = errors.New("tag has associated bookmarks")
)

type TagService struct {
	repo         *repositories.TagRepository
	cacheService *CacheService
}

func NewTagService() *TagService {
	return &TagService{
		repo:         repositories.NewTagRepository(),
		cacheService: &CacheService{},
	}
}

func (s *TagService) FindByUserIDService(userId string) ([]models.TagWithCount, error) {
	ctx := context.Background()

	// CACHE HIT PATH: Intentar obtener desde caché
	cachedTags, hit, err := s.cacheService.GetTagsFromCache(ctx, userId)
	if hit && err == nil {
		return cachedTags, nil
	}

	// CACHE MISS PATH: Consultar base de datos
	tags, err := s.repo.FindByUserIDWithCount(userId)

	if err != nil {
		return nil, err
	}

	// Guardar en caché para futuras consultas
	// No retornamos error si el caché falla - la app continúa funcionando
	_ = s.cacheService.SetTagsCache(ctx, userId, tags)

	return tags, nil
}

func (s *TagService) FindOrCreateTags(tagNames []string, userID string) ([]models.Tag, error) {
	var tags []models.Tag
	tagMap := make(map[string]models.Tag)

	for _, tagName := range tagNames {

		tagName = strings.TrimSpace(tagName)

		if tagName == "" {
			continue
		}

		if existingTag, exists := tagMap[tagName]; exists {
			tags = append(tags, existingTag)
			continue
		}

		tag, err := s.repo.FindByTitleAndUserID(tagName, userID)

		if err == gorm.ErrRecordNotFound {
			newTag := models.Tag{
				Title:  tagName,
				UserID: userID,
			}

			if err := s.repo.Create(&newTag); err != nil {
				return nil, err
			}

			tag = &newTag
		} else if err != nil {
			return nil, err
		}

		tagMap[tagName] = *tag
		tags = append(tags, *tag)
	}

	return tags, nil
}

func (s *TagService) CreateTagService(title string, userID string) (*models.Tag, error) {
	ctx := context.Background()

	// Verificar si ya existe un tag con ese título para este usuario
	existingTag, err := s.repo.FindByTitleAndUserID(title, userID)

	if err == nil && existingTag != nil {
		return nil, ErrTagAlreadyExists
	}

	// Si el error no es "record not found", es un error real
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// Crear el nuevo tag
	newTag := models.Tag{
		Title:  strings.TrimSpace(title),
		UserID: userID,
	}

	if err := s.repo.Create(&newTag); err != nil {
		return nil, err
	}

	// Invalidar caché de tags del usuario
	_ = s.cacheService.InvalidateTagsCache(ctx, userID)

	return &newTag, nil
}

func (s *TagService) UpdateTagService(tagID string, title string, userID string) (*models.Tag, error) {
	ctx := context.Background()

	// Buscar el tag por ID
	tag, err := s.repo.FindByID(tagID)

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrTagNotFound
		}
		return nil, err
	}

	// Verificar que el tag pertenece al usuario
	if tag.UserID != userID {
		return nil, ErrTagUnauthorized
	}

	// Verificar que no exista otro tag con el mismo título para este usuario
	existingTag, err := s.repo.FindByTitleAndUserID(title, userID)

	if err == nil && existingTag != nil && existingTag.ID != tag.ID {
		return nil, ErrTagAlreadyExists
	}

	// Si el error no es "record not found", es un error real
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// Actualizar el título
	tag.Title = strings.TrimSpace(title)

	if err := s.repo.Update(tag); err != nil {
		return nil, err
	}

	// Invalidar caché de tags del usuario
	_ = s.cacheService.InvalidateTagsCache(ctx, userID)

	return tag, nil
}

func (s *TagService) DeleteTagService(tagID string, userID string) error {
	ctx := context.Background()

	// Buscar el tag por ID
	tag, err := s.repo.FindByID(tagID)

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrTagNotFound
		}
		return err
	}

	// Verificar que el tag pertenece al usuario
	if tag.UserID != userID {
		return ErrTagUnauthorized
	}

	// Verificar que el tag no tenga bookmarks asociados
	count, err := s.repo.CountBookmarks(tag.ID)

	if err != nil {
		return err
	}

	if count > 0 {
		return ErrTagHasBookmarks
	}

	// Eliminar el tag (soft delete)
	if err := s.repo.Delete(tag); err != nil {
		return err
	}

	// Invalidar caché de tags del usuario
	_ = s.cacheService.InvalidateTagsCache(ctx, userID)

	return nil
}
