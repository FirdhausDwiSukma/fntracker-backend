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
