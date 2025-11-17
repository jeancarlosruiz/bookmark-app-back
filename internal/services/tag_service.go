package services

import (
	"strings"

	"github.com/jeancarlosruiz/bookmark-app-back/internal/models"
	"github.com/jeancarlosruiz/bookmark-app-back/internal/repositories"
	"gorm.io/gorm"
)

type TagService struct {
	repo *repositories.TagRepository
}

func NewTagService() *TagService {
	return &TagService{
		repo: repositories.NewTagRepository(),
	}
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
