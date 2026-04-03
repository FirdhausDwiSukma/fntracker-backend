package dto

type BudgetRequest struct {
	CategoryID  uint    `json:"category_id" binding:"required"`
	LimitAmount float64 `json:"limit_amount" binding:"required,gt=0"`
	Month       int     `json:"month" binding:"required,min=1,max=12"`
	Year        int     `json:"year" binding:"required,min=2000,max=2100"`
}

type BudgetResponse struct {
	ID          uint    `json:"id"`
	CategoryID  uint    `json:"category_id"`
	Category    string  `json:"category_name"`
	LimitAmount float64 `json:"limit_amount"`
	UsedAmount  float64 `json:"used_amount"`
	Percentage  float64 `json:"percentage"`
	Warning     bool    `json:"warning"`  // >= 80%
	Exceeded    bool    `json:"exceeded"` // >= 100%
	Month       int     `json:"month"`
	Year        int     `json:"year"`
}
