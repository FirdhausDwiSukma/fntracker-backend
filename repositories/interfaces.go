package repositories

import (
	"finance-tracker/dto"
	"finance-tracker/models"
)

type UserRepository interface {
	Create(user *models.User) error
	FindByEmail(email string) (*models.User, error)
	FindByID(id uint) (*models.User, error)
}

type CategoryRepository interface {
	FindAllByUser(userID uint) ([]models.Category, error)
	FindByIDAndUser(id, userID uint) (*models.Category, error)
	Create(category *models.Category) error
	Update(category *models.Category) error
	Delete(id, userID uint) error
	ExistsByNameTypeUser(name, categoryType string, userID uint) (bool, error)
}

type TransactionRepository interface {
	FindAllByUser(userID uint, filter dto.TransactionFilter) ([]models.Transaction, int64, error)
	FindByIDAndUser(id, userID uint) (*models.Transaction, error)
	Create(tx *models.Transaction) error
	Update(tx *models.Transaction) error
	Delete(id, userID uint) error
	SumByCategory(userID uint, categoryID uint, month, year int) (float64, error)
	GetMonthlyAggregates(userID uint, months int) ([]dto.MonthlyAggregate, error)
	GetTopExpenseCategories(userID uint, month, year int, limit int) ([]dto.CategoryExpense, error)
}

type BudgetRepository interface {
	FindAllByUser(userID uint, month, year int) ([]models.Budget, error)
	FindByIDAndUser(id, userID uint) (*models.Budget, error)
	FindByUserCategoryMonthYear(userID, categoryID uint, month, year int) (*models.Budget, error)
	Create(budget *models.Budget) error
	Update(budget *models.Budget) error
	Delete(id, userID uint) error
}
