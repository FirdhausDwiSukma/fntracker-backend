package services

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"time"

	"finance-tracker/dto"
	"finance-tracker/models"
	"finance-tracker/repositories"
	"finance-tracker/utils"
)

type transactionService struct {
	txRepo       repositories.TransactionRepository
	categoryRepo repositories.CategoryRepository
}

func NewTransactionService(txRepo repositories.TransactionRepository, categoryRepo repositories.CategoryRepository) TransactionService {
	return &transactionService{txRepo: txRepo, categoryRepo: categoryRepo}
}

func (s *transactionService) GetAllByUser(userID uint, filter dto.TransactionFilter) ([]models.Transaction, int64, error) {
	return s.txRepo.FindAllByUser(userID, filter)
}

func (s *transactionService) Create(userID uint, req dto.TransactionRequest) (*models.Transaction, error) {
	// Validate category ownership
	cat, err := s.categoryRepo.FindByIDAndUser(req.CategoryID, userID)
	if err != nil {
		return nil, err
	}
	if cat == nil {
		return nil, errors.New("invalid category")
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, errors.New("invalid date format, expected YYYY-MM-DD")
	}

	tx := &models.Transaction{
		UserID:      userID,
		CategoryID:  req.CategoryID,
		Amount:      req.Amount,
		Type:        req.Type,
		Description: utils.SanitizeString(req.Description),
		Date:        date,
	}

	if err := s.txRepo.Create(tx); err != nil {
		return nil, err
	}

	// Reload with category preloaded
	return s.txRepo.FindByIDAndUser(tx.ID, userID)
}

func (s *transactionService) Update(userID uint, txID uint, req dto.TransactionRequest) (*models.Transaction, error) {
	existing, err := s.txRepo.FindByIDAndUser(txID, userID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, errors.New("not found")
	}

	// Validate category ownership
	cat, err := s.categoryRepo.FindByIDAndUser(req.CategoryID, userID)
	if err != nil {
		return nil, err
	}
	if cat == nil {
		return nil, errors.New("invalid category")
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, errors.New("invalid date format, expected YYYY-MM-DD")
	}

	existing.CategoryID = req.CategoryID
	existing.Amount = req.Amount
	existing.Type = req.Type
	existing.Description = utils.SanitizeString(req.Description)
	existing.Date = date

	if err := s.txRepo.Update(existing); err != nil {
		return nil, err
	}

	return s.txRepo.FindByIDAndUser(existing.ID, userID)
}

func (s *transactionService) Delete(userID uint, txID uint) error {
	existing, err := s.txRepo.FindByIDAndUser(txID, userID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("not found")
	}
	return s.txRepo.Delete(txID, userID)
}

func (s *transactionService) ExportCSV(userID uint, filter dto.ExportFilter) ([]byte, error) {
	txFilter := dto.TransactionFilter{
		StartDate: filter.StartDate,
		EndDate:   filter.EndDate,
		Page:      1,
		Limit:     100000, // large limit for export
	}

	transactions, _, err := s.txRepo.FindAllByUser(userID, txFilter)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	// Write header
	if err := w.Write([]string{"ID", "Date", "Type", "Category", "Amount", "Description"}); err != nil {
		return nil, err
	}

	for _, tx := range transactions {
		record := []string{
			fmt.Sprintf("%d", tx.ID),
			tx.Date.Format("2006-01-02"),
			tx.Type,
			tx.Category.Name,
			fmt.Sprintf("%.2f", tx.Amount),
			tx.Description,
		}
		if err := w.Write(record); err != nil {
			return nil, err
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
