package services

import (
	"math"
	"time"

	"finance-tracker/dto"
	"finance-tracker/repositories"
)

type dashboardService struct {
	transactionRepo repositories.TransactionRepository
	budgetRepo      repositories.BudgetRepository
}

func NewDashboardService(
	transactionRepo repositories.TransactionRepository,
	budgetRepo repositories.BudgetRepository,
) DashboardService {
	return &dashboardService{
		transactionRepo: transactionRepo,
		budgetRepo:      budgetRepo,
	}
}

func (s *dashboardService) GetSummary(userID uint, month, year int) (*dto.DashboardResponse, error) {
	// Default to current month/year if not provided
	now := time.Now()
	if month == 0 {
		month = int(now.Month())
	}
	if year == 0 {
		year = now.Year()
	}

	// Get total income for the month
	totalIncome, err := s.sumByType(userID, "income", month, year)
	if err != nil {
		return nil, err
	}

	// Get total expense for the month
	totalExpense, err := s.sumByType(userID, "expense", month, year)
	if err != nil {
		return nil, err
	}

	balance := totalIncome - totalExpense

	// Get 12-month aggregates
	monthlyData, err := s.transactionRepo.GetMonthlyAggregates(userID, 12)
	if err != nil {
		return nil, err
	}
	if monthlyData == nil {
		monthlyData = []dto.MonthlyAggregate{}
	}

	// Get top 5 expense categories for the month
	topCategories, err := s.transactionRepo.GetTopExpenseCategories(userID, month, year, 5)
	if err != nil {
		return nil, err
	}
	if topCategories == nil {
		topCategories = []dto.CategoryExpense{}
	}

	// Get budget status for the month
	budgets, err := s.budgetRepo.FindAllByUser(userID, month, year)
	if err != nil {
		return nil, err
	}

	budgetStatus := make([]dto.BudgetResponse, 0, len(budgets))
	for _, b := range budgets {
		used, err := s.transactionRepo.SumByCategory(userID, b.CategoryID, b.Month, b.Year)
		if err != nil {
			return nil, err
		}

		percentage := 0.0
		if b.LimitAmount > 0 {
			percentage = math.Round((used/b.LimitAmount)*10000) / 100
		}

		budgetStatus = append(budgetStatus, dto.BudgetResponse{
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

	return &dto.DashboardResponse{
		TotalIncome:   totalIncome,
		TotalExpense:  totalExpense,
		Balance:       balance,
		MonthlyData:   monthlyData,
		TopCategories: topCategories,
		BudgetStatus:  budgetStatus,
	}, nil
}

// sumByType aggregates transaction amounts for a given type, month, and year.
// Returns 0 (not null) when no transactions exist for the period.
func (s *dashboardService) sumByType(userID uint, txType string, month, year int) (float64, error) {
	transactions, _, err := s.transactionRepo.FindAllByUser(userID, dto.TransactionFilter{
		Type:  txType,
		Month: month,
		Year:  year,
		Page:  1,
		Limit: 100000,
	})
	if err != nil {
		return 0, err
	}

	var total float64
	for _, tx := range transactions {
		total += tx.Amount
	}
	return total, nil
}
