package services

import (
	"context"
	"strings"

	"github.com/jeancarlosruiz/bookmark-app-back/internal/models"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/repositories"
	"gorm.io/gorm"
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
