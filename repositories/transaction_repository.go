package repositories

import (
	"errors"
	"fmt"
	"time"

	"finance-tracker/dto"
	"finance-tracker/models"

	"gorm.io/gorm"
)

type transactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) FindAllByUser(userID uint, filter dto.TransactionFilter) ([]models.Transaction, int64, error) {
	query := r.db.Model(&models.Transaction{}).
		Preload("Category").
		Where("transactions.user_id = ?", userID)

	if filter.Type != "" {
		query = query.Where("transactions.type = ?", filter.Type)
	}
	if filter.CategoryID != 0 {
		query = query.Where("transactions.category_id = ?", filter.CategoryID)
	}
	if filter.StartDate != "" {
		query = query.Where("transactions.date >= ?", filter.StartDate)
	}
	if filter.EndDate != "" {
		query = query.Where("transactions.date <= ?", filter.EndDate)
	}
	if filter.Month != 0 {
		query = query.Where("EXTRACT(MONTH FROM transactions.date) = ?", filter.Month)
	}
	if filter.Year != 0 {
		query = query.Where("EXTRACT(YEAR FROM transactions.date) = ?", filter.Year)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	var transactions []models.Transaction
	if err := query.Order("transactions.date DESC, transactions.id DESC").
		Limit(limit).Offset(offset).
		Find(&transactions).Error; err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}

func (r *transactionRepository) FindByIDAndUser(id, userID uint) (*models.Transaction, error) {
	var tx models.Transaction
	result := r.db.Preload("Category").
		Where("id = ? AND user_id = ?", id, userID).
		First(&tx)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &tx, nil
}

func (r *transactionRepository) Create(tx *models.Transaction) error {
	return r.db.Create(tx).Error
}

func (r *transactionRepository) Update(tx *models.Transaction) error {
	return r.db.Save(tx).Error
}

func (r *transactionRepository) Delete(id, userID uint) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Transaction{}).Error
}

func (r *transactionRepository) SumByCategory(userID uint, categoryID uint, month, year int) (float64, error) {
	var total float64
	err := r.db.Model(&models.Transaction{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("user_id = ? AND category_id = ? AND EXTRACT(MONTH FROM date) = ? AND EXTRACT(YEAR FROM date) = ?",
			userID, categoryID, month, year).
		Scan(&total).Error
	return total, err
}

func (r *transactionRepository) GetMonthlyAggregates(userID uint, months int) ([]dto.MonthlyAggregate, error) {
	type row struct {
		Year    int
		Month   int
		Income  float64
		Expense float64
	}

	var rows []row
	err := r.db.Model(&models.Transaction{}).
		Select(`
			EXTRACT(YEAR FROM date)::int AS year,
			EXTRACT(MONTH FROM date)::int AS month,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) AS income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS expense
		`).
		Where("user_id = ? AND date >= ?", userID, time.Now().AddDate(0, -months+1, 0).Format("2006-01-02")).
		Group("year, month").
		Order("year ASC, month ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]dto.MonthlyAggregate, 0, len(rows))
	for _, r := range rows {
		label := fmt.Sprintf("%s %d", time.Month(r.Month).String()[:3], r.Year)
		result = append(result, dto.MonthlyAggregate{
			Month:   label,
			Income:  r.Income,
			Expense: r.Expense,
		})
	}
	return result, nil
}

func (r *transactionRepository) GetTopExpenseCategories(userID uint, month, year int, limit int) ([]dto.CategoryExpense, error) {
	type row struct {
		CategoryID   uint
		CategoryName string
		Total        float64
	}

	var rows []row
	query := r.db.Model(&models.Transaction{}).
		Select("transactions.category_id, categories.name AS category_name, COALESCE(SUM(transactions.amount), 0) AS total").
		Joins("JOIN categories ON categories.id = transactions.category_id").
		Where("transactions.user_id = ? AND transactions.type = 'expense'", userID)

	if month != 0 {
		query = query.Where("EXTRACT(MONTH FROM transactions.date) = ?", month)
	}
	if year != 0 {
		query = query.Where("EXTRACT(YEAR FROM transactions.date) = ?", year)
	}

	if err := query.Group("transactions.category_id, categories.name").
		Order("total DESC").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	// Calculate total for percentage
	var grandTotal float64
	for _, r := range rows {
		grandTotal += r.Total
	}

	result := make([]dto.CategoryExpense, 0, len(rows))
	for _, r := range rows {
		pct := 0.0
		if grandTotal > 0 {
			pct = (r.Total / grandTotal) * 100
		}
		result = append(result, dto.CategoryExpense{
			CategoryID:   r.CategoryID,
			CategoryName: r.CategoryName,
			Total:        r.Total,
			Percentage:   pct,
		})
	}
	return result, nil
}
