package services

import (
	"finance-tracker/dto"
	"finance-tracker/models"
)

// LoginResponse holds the data returned after a successful login.
type LoginResponse struct {
	User      *models.User
	Token     string
	CsrfToken string
}

type AuthService interface {
	Register(req dto.RegisterRequest) (*models.User, error)
	Login(req dto.LoginRequest) (*LoginResponse, error)
}

type CategoryService interface {
	GetAllByUser(userID uint) ([]models.Category, error)
	Create(userID uint, req dto.CategoryRequest) (*models.Category, error)
	Update(userID uint, categoryID uint, req dto.CategoryRequest) (*models.Category, error)
	Delete(userID uint, categoryID uint) error
}

type TransactionService interface {
	GetAllByUser(userID uint, filter dto.TransactionFilter) ([]models.Transaction, int64, error)
	Create(userID uint, req dto.TransactionRequest) (*models.Transaction, error)
	Update(userID uint, txID uint, req dto.TransactionRequest) (*models.Transaction, error)
	Delete(userID uint, txID uint) error
	ExportCSV(userID uint, filter dto.ExportFilter) ([]byte, error)
}

type BudgetService interface {
	GetAllByUser(userID uint, month, year int) ([]dto.BudgetResponse, error)
	Create(userID uint, req dto.BudgetRequest) (*models.Budget, error)
	Update(userID uint, budgetID uint, req dto.BudgetRequest) (*models.Budget, error)
	Delete(userID uint, budgetID uint) error
}
