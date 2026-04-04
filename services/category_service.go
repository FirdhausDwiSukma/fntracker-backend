package services

import (
	"errors"

	"finance-tracker/dto"
	"finance-tracker/models"
	"finance-tracker/repositories"
	"finance-tracker/utils"
)

type categoryService struct {
	categoryRepo repositories.CategoryRepository
}

func NewCategoryService(categoryRepo repositories.CategoryRepository) CategoryService {
	return &categoryService{categoryRepo: categoryRepo}
}

func (s *categoryService) GetAllByUser(userID uint) ([]models.Category, error) {
	return s.categoryRepo.FindAllByUser(userID)
}

func (s *categoryService) Create(userID uint, req dto.CategoryRequest) (*models.Category, error) {
	name := utils.SanitizeString(req.Name)

	exists, err := s.categoryRepo.ExistsByNameTypeUser(name, req.Type, userID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("category already exists")
	}

	category := &models.Category{
		UserID: userID,
		Name:   name,
		Type:   req.Type,
	}

	if err := s.categoryRepo.Create(category); err != nil {
		return nil, err
	}

	return category, nil
}

func (s *categoryService) Update(userID uint, categoryID uint, req dto.CategoryRequest) (*models.Category, error) {
	category, err := s.categoryRepo.FindByIDAndUser(categoryID, userID)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, errors.New("not found")
	}

	name := utils.SanitizeString(req.Name)

	// Check uniqueness only if name or type changed
	if name != category.Name || req.Type != category.Type {
		exists, err := s.categoryRepo.ExistsByNameTypeUser(name, req.Type, userID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("category already exists")
		}
	}

	category.Name = name
	category.Type = req.Type

	if err := s.categoryRepo.Update(category); err != nil {
		return nil, err
	}

	return category, nil
}

func (s *categoryService) Delete(userID uint, categoryID uint) error {
	category, err := s.categoryRepo.FindByIDAndUser(categoryID, userID)
	if err != nil {
		return err
	}
	if category == nil {
		return errors.New("not found")
	}

	return s.categoryRepo.Delete(categoryID, userID)
}
