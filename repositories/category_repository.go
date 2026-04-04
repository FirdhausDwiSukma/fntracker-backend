package repositories

import (
	"errors"

	"finance-tracker/models"

	"gorm.io/gorm"
)

type categoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) CategoryRepository {
	return &categoryRepository{db: db}
}

func (r *categoryRepository) FindAllByUser(userID uint) ([]models.Category, error) {
	var categories []models.Category
	if err := r.db.Where("user_id = ?", userID).Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *categoryRepository) FindByIDAndUser(id, userID uint) (*models.Category, error) {
	var category models.Category
	result := r.db.Where("id = ? AND user_id = ?", id, userID).First(&category)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &category, nil
}

func (r *categoryRepository) Create(category *models.Category) error {
	return r.db.Create(category).Error
}

func (r *categoryRepository) Update(category *models.Category) error {
	return r.db.Save(category).Error
}

func (r *categoryRepository) Delete(id, userID uint) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Category{}).Error
}

func (r *categoryRepository) ExistsByNameTypeUser(name, categoryType string, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.Category{}).
		Where("name = ? AND type = ? AND user_id = ?", name, categoryType, userID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
