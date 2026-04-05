package dto

import "finance-tracker/models"

type TransactionRequest struct {
	CategoryID  uint    `json:"category_id" binding:"required"`
	Amount      float64 `json:"amount" binding:"required,gt=0"`
	Type        string  `json:"type" binding:"required,oneof=income expense"`
	Description string  `json:"description" binding:"max=500"`
	Date        string  `json:"date" binding:"required"` // ISO 8601: YYYY-MM-DD
}

type TransactionResponse struct {
	ID           uint    `json:"id"`
	CategoryID   uint    `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Amount       float64 `json:"amount"`
	Type         string  `json:"type"`
	Description  string  `json:"description"`
	Date         string  `json:"date"`
}

func ToTransactionResponse(tx models.Transaction) TransactionResponse {
	return TransactionResponse{
		ID:           tx.ID,
		CategoryID:   tx.CategoryID,
		CategoryName: tx.Category.Name,
		Amount:       tx.Amount,
		Type:         tx.Type,
		Description:  tx.Description,
		Date:         tx.Date.Format("2006-01-02"),
	}
}

func ToTransactionResponsePtr(tx *models.Transaction) TransactionResponse {
	if tx == nil {
		return TransactionResponse{}
	}
	return ToTransactionResponse(*tx)
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
