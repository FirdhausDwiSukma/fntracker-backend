package dto

type TransactionRequest struct {
	CategoryID  uint    `json:"category_id" binding:"required"`
	Amount      float64 `json:"amount" binding:"required,gt=0"`
	Type        string  `json:"type" binding:"required,oneof=income expense"`
	Description string  `json:"description" binding:"max=500"`
	Date        string  `json:"date" binding:"required"` // ISO 8601: YYYY-MM-DD
}

type TransactionFilter struct {
	Type       string
	CategoryID uint
	StartDate  string
	EndDate    string
	Month      int
	Year       int
	Page       int
	Limit      int
}

type ExportFilter struct {
	StartDate string
	EndDate   string
}
