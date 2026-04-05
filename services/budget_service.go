package services

import (
	"errors"
	"math"

	"finance-tracker/dto"
	"finance-tracker/models"
	"finance-tracker/repositories"
)

type budgetService struct {
	budgetRepo      repositories.BudgetRepository
	transactionRepo repositories.TransactionRepository
	categoryRepo    repositories.CategoryRepository
}

func NewBudgetService(
	budgetRepo repositories.BudgetRepository,
	transactionRepo repositories.TransactionRepository,
	categoryRepo repositories.CategoryRepository,
) BudgetService {
	return &budgetService{
		budgetRepo:      budgetRepo,
		transactionRepo: transactionRepo,
		categoryRepo:    categoryRepo,
	}
}

func (s *budgetService) GetAllByUser(userID uint, month, year int) ([]dto.BudgetResponse, error) {
	budgets, err := s.budgetRepo.FindAllByUser(userID, month, year)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.BudgetResponse, 0, len(budgets))
	for _, b := range budgets {
		used, err := s.transactionRepo.SumByCategory(userID, b.CategoryID, b.Month, b.Year)
		if err != nil {
			return nil, err
		}

		percentage := 0.0
		if b.LimitAmount > 0 {
			percentage = math.Round((used/b.LimitAmount)*10000) / 100 // 2 decimal places
		}

		responses = append(responses, dto.BudgetResponse{
			ID:          b.ID,
			CategoryID:  b.CategoryID,
			Category:    b.Category.Name,
			LimitAmount: b.LimitAmount,
			UsedAmount:  used,
			Percentage:  percentage,
			Warning:     percentage >= 80,
			Exceeded:    percentage >= 100,
			Month:       b.Month,
			Year:        b.Year,
		})
	}

	return responses, nil
}

func (s *budgetService) Create(userID uint, req dto.BudgetRequest) (*models.Budget, error) {
	// Verify category belongs to user
	category, err := s.categoryRepo.FindByIDAndUser(req.CategoryID, userID)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, errors.New("category not found")
	}

	// Enforce uniqueness: one budget per user/category/month/year
	existing, err := s.budgetRepo.FindByUserCategoryMonthYear(userID, req.CategoryID, req.Month, req.Year)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("budget already exists")
	}

	budget := &models.Budget{
		UserID:      userID,
		CategoryID:  req.CategoryID,
		LimitAmount: req.LimitAmount,
		Month:       req.Month,
		Year:        req.Year,
	}

	if err := s.budgetRepo.Create(budget); err != nil {
		return nil, err
	}

	return budget, nil
}

func (s *budgetService) Update(userID uint, budgetID uint, req dto.BudgetRequest) (*models.Budget, error) {
	budget, err := s.budgetRepo.FindByIDAndUser(budgetID, userID)
	if err != nil {
		return nil, err
	}
	if budget == nil {
		return nil, errors.New("not found")
	}

	// Verify category belongs to user
	category, err := s.categoryRepo.FindByIDAndUser(req.CategoryID, userID)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, errors.New("category not found")
	}

	// Check uniqueness if key fields changed
	if budget.CategoryID != req.CategoryID || budget.Month != req.Month || budget.Year != req.Year {
		existing, err := s.budgetRepo.FindByUserCategoryMonthYear(userID, req.CategoryID, req.Month, req.Year)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != budgetID {
			return nil, errors.New("budget already exists")
		}
	}

	budget.CategoryID = req.CategoryID
	budget.LimitAmount = req.LimitAmount
	budget.Month = req.Month
	budget.Year = req.Year

	if err := s.budgetRepo.Update(budget); err != nil {
		return nil, err
	}

	return budget, nil
}

func (s *budgetService) Delete(userID uint, budgetID uint) error {
	budget, err := s.budgetRepo.FindByIDAndUser(budgetID, userID)
	if err != nil {
		return err
	}
	if budget == nil {
		return errors.New("not found")
	}

	return s.budgetRepo.Delete(budgetID, userID)
}
