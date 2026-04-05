package repositories

import (
	"errors"

	"finance-tracker/models"

	"gorm.io/gorm"
)

type budgetRepository struct {
	db *gorm.DB
}

func NewBudgetRepository(db *gorm.DB) BudgetRepository {
	return &budgetRepository{db: db}
}

func (r *budgetRepository) FindAllByUser(userID uint, month, year int) ([]models.Budget, error) {
	var budgets []models.Budget
	query := r.db.Preload("Category").Where("user_id = ?", userID)
	if month > 0 {
		query = query.Where("month = ?", month)
	}
	if year > 0 {
		query = query.Where("year = ?", year)
	}
	if err := query.Find(&budgets).Error; err != nil {
		return nil, err
	}
	return budgets, nil
}

func (r *budgetRepository) FindByIDAndUser(id, userID uint) (*models.Budget, error) {
	var budget models.Budget
	result := r.db.Preload("Category").Where("id = ? AND user_id = ?", id, userID).First(&budget)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &budget, nil
}

func (r *budgetRepository) FindByUserCategoryMonthYear(userID, categoryID uint, month, year int) (*models.Budget, error) {
	var budget models.Budget
	result := r.db.Where("user_id = ? AND category_id = ? AND month = ? AND year = ?",
		userID, categoryID, month, year).First(&budget)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &budget, nil
}

func (r *budgetRepository) Create(budget *models.Budget) error {
	return r.db.Create(budget).Error
}

func (r *budgetRepository) Update(budget *models.Budget) error {
	return r.db.Save(budget).Error
}

func (r *budgetRepository) Delete(id, userID uint) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Budget{}).Error
}
