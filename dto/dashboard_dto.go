package dto

type DashboardResponse struct {
	TotalIncome   float64            `json:"total_income"`
	TotalExpense  float64            `json:"total_expense"`
	Balance       float64            `json:"balance"`
	MonthlyData   []MonthlyAggregate `json:"monthly_data"`
	TopCategories []CategoryExpense  `json:"top_categories"`
	BudgetStatus  []BudgetResponse   `json:"budget_status"`
}

type MonthlyAggregate struct {
	Month   string  `json:"month"` // e.g. "Jan 2024"
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
}

type CategoryExpense struct {
	CategoryID   uint    `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Total        float64 `json:"total"`
	Percentage   float64 `json:"percentage"`
}
