package services

import (
	"errors"

	"finance-tracker/dto"
	"finance-tracker/models"
	"finance-tracker/repositories"
	"finance-tracker/utils"

	"golang.org/x/crypto/bcrypt"
)

type authService struct {
	userRepo  repositories.UserRepository
	jwtSecret string
}

func NewAuthService(userRepo repositories.UserRepository, jwtSecret string) AuthService {
	return &authService{userRepo: userRepo, jwtSecret: jwtSecret}
}

func (s *authService) Register(req dto.RegisterRequest) (*models.User, error) {
	existing, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("email already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Name:     utils.SanitizeString(req.Name),
		Email:    req.Email,
		Password: string(hash),
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	user.Password = ""
	return user, nil
}

func (s *authService) Login(req dto.LoginRequest) (*LoginResponse, error) {
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	token, err := utils.GenerateToken(user.ID, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	csrfToken, err := utils.GenerateCsrfToken()
	if err != nil {
		return nil, err
	}

	user.Password = ""
	return &LoginResponse{User: user, Token: token, CsrfToken: csrfToken}, nil
}
