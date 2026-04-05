// Package services_test contains unit tests for TransactionService.
// Validates: Requirements 5.1–5.8, 11.1–11.3
package services_test

import (
	"encoding/csv"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"finance-tracker/dto"
	"finance-tracker/models"
	"finance-tracker/services"
)

// ─── Mock TransactionRepository ──────────────────────────────────────────────

type mockTransactionRepo struct {
	transactions   []*models.Transaction
	nextID         uint
	forceCreateErr error
	forceUpdateErr error
	forceDeleteErr error
	forceFindErr   error
}

func newMockTransactionRepo() *mockTransactionRepo {
	return &mockTransactionRepo{nextID: 1}
}

func (m *mockTransactionRepo) FindAllByUser(userID uint, filter dto.TransactionFilter) ([]models.Transaction, int64, error) {
	if m.forceFindErr != nil {
		return nil, 0, m.forceFindErr
	}
	var result []models.Transaction
	for _, tx := range m.transactions {
		if tx.UserID != userID {
			continue
		}
		if filter.Type != "" && tx.Type != filter.Type {
			continue
		}
		if filter.CategoryID != 0 && tx.CategoryID != filter.CategoryID {
			continue
		}
		if filter.StartDate != "" {
			start, err := time.Parse("2006-01-02", filter.StartDate)
			if err == nil && tx.Date.Before(start) {
				continue
			}
		}
		if filter.EndDate != "" {
			end, err := time.Parse("2006-01-02", filter.EndDate)
			if err == nil && tx.Date.After(end) {
				continue
			}
		}
		if filter.Month != 0 && int(tx.Date.Month()) != filter.Month {
			continue
		}
		if filter.Year != 0 && tx.Date.Year() != filter.Year {
			continue
		}
		result = append(result, *tx)
	}
	return result, int64(len(result)), nil
}

func (m *mockTransactionRepo) FindByIDAndUser(id, userID uint) (*models.Transaction, error) {
	if m.forceFindErr != nil {
		return nil, m.forceFindErr
	}
	for _, tx := range m.transactions {
		if tx.ID == id && tx.UserID == userID {
			cp := *tx
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockTransactionRepo) Create(tx *models.Transaction) error {
	if m.forceCreateErr != nil {
		return m.forceCreateErr
	}
	tx.ID = m.nextID
	m.nextID++
	cp := *tx
	m.transactions = append(m.transactions, &cp)
	return nil
}

func (m *mockTransactionRepo) Update(tx *models.Transaction) error {
	if m.forceUpdateErr != nil {
		return m.forceUpdateErr
	}
	for i, t := range m.transactions {
		if t.ID == tx.ID {
			cp := *tx
			m.transactions[i] = &cp
			return nil
		}
	}
	return errors.New("not found")
}

func (m *mockTransactionRepo) Delete(id, userID uint) error {
	if m.forceDeleteErr != nil {
		return m.forceDeleteErr
	}
	for i, tx := range m.transactions {
		if tx.ID == id && tx.UserID == userID {
			m.transactions = append(m.transactions[:i], m.transactions[i+1:]...)
			return nil
		}
	}
	return errors.New("not found")
}

func (m *mockTransactionRepo) SumByCategory(userID uint, categoryID uint, month, year int) (float64, error) {
	return 0, nil
}

func (m *mockTransactionRepo) GetMonthlyAggregates(userID uint, months int) ([]dto.MonthlyAggregate, error) {
	return nil, nil
}

func (m *mockTransactionRepo) GetTopExpenseCategories(userID uint, month, year int, limit int) ([]dto.CategoryExpense, error) {
	return nil, nil
}

// ─── Mock CategoryRepository (reuse pattern from category_service_test.go) ───

type mockCatRepo struct {
	categories   []*models.Category
	nextID       uint
	forceFindErr error
}

func newMockCatRepo() *mockCatRepo {
	return &mockCatRepo{nextID: 1}
}

func (m *mockCatRepo) FindAllByUser(userID uint) ([]models.Category, error) {
	var result []models.Category
	for _, c := range m.categories {
		if c.UserID == userID {
			result = append(result, *c)
		}
	}
	return result, nil
}

func (m *mockCatRepo) FindByIDAndUser(id, userID uint) (*models.Category, error) {
	if m.forceFindErr != nil {
		return nil, m.forceFindErr
	}
	for _, c := range m.categories {
		if c.ID == id && c.UserID == userID {
			cp := *c
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockCatRepo) Create(category *models.Category) error {
	category.ID = m.nextID
	m.nextID++
	cp := *category
	m.categories = append(m.categories, &cp)
	return nil
}

func (m *mockCatRepo) Update(category *models.Category) error { return nil }

func (m *mockCatRepo) Delete(id, userID uint) error { return nil }

func (m *mockCatRepo) ExistsByNameTypeUser(name, categoryType string, userID uint) (bool, error) {
	return false, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func newTransactionService(txRepo *mockTransactionRepo, catRepo *mockCatRepo) services.TransactionService {
	return services.NewTransactionService(txRepo, catRepo)
}

// seedCat adds a category directly to the mock repo and returns it.
func seedCat(repo *mockCatRepo, userID uint, name, typ string) *models.Category {
	cat := &models.Category{UserID: userID, Name: name, Type: typ}
	_ = repo.Create(cat)
	return cat
}

// seedTx creates a transaction via the service and returns it.
func seedTx(t *testing.T, svc services.TransactionService, userID uint, catID uint, amount float64, typ, date, desc string) *models.Transaction {
	t.Helper()
	tx, err := svc.Create(userID, dto.TransactionRequest{
		CategoryID:  catID,
		Amount:      amount,
		Type:        typ,
		Date:        date,
		Description: desc,
	})
	if err != nil {
		t.Fatalf("seedTx failed: %v", err)
	}
	return tx
}

// ─── GetAllByUser (Req 5.1) ───────────────────────────────────────────────────

// Req 5.1: GetAllByUser returns only the authenticated user's transactions.
func TestGetAllByUser_ReturnsOnlyOwnTransactions(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	cat1 := seedCat(catRepo, 1, "Salary", "income")
	cat2 := seedCat(catRepo, 2, "Salary", "income")

	seedTx(t, svc, 1, cat1.ID, 1000, "income", "2024-01-15", "user1 tx")
	seedTx(t, svc, 2, cat2.ID, 500, "income", "2024-01-15", "user2 tx")

	txs, total, err := svc.GetAllByUser(1, dto.TransactionFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 transaction for user 1, got %d", total)
	}
	if len(txs) != 1 {
		t.Fatalf("expected 1 transaction slice, got %d", len(txs))
	}
	if txs[0].UserID != 1 {
		t.Errorf("returned transaction belongs to user %d, not user 1", txs[0].UserID)
	}
}

func TestGetAllTransactions_EmptyForNewUser(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	txs, total, err := svc.GetAllByUser(99, dto.TransactionFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 || len(txs) != 0 {
		t.Fatalf("expected 0 transactions, got %d", total)
	}
}

// ─── Create (Req 5.2, 5.3, 5.4) ──────────────────────────────────────────────

// Req 5.2: Create with valid data saves transaction with correct fields.
func TestCreate_ValidData(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	cat := seedCat(catRepo, 1, "Food", "expense")

	tx, err := svc.Create(1, dto.TransactionRequest{
		CategoryID:  cat.ID,
		Amount:      150.50,
		Type:        "expense",
		Date:        "2024-03-10",
		Description: "Groceries",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tx == nil {
		t.Fatal("expected transaction, got nil")
	}
	if tx.UserID != 1 {
		t.Errorf("UserID: got %d, want 1", tx.UserID)
	}
	if tx.CategoryID != cat.ID {
		t.Errorf("CategoryID: got %d, want %d", tx.CategoryID, cat.ID)
	}
	if tx.Amount != 150.50 {
		t.Errorf("Amount: got %.2f, want 150.50", tx.Amount)
	}
	if tx.Type != "expense" {
		t.Errorf("Type: got %q, want expense", tx.Type)
	}
	if tx.Date.Format("2006-01-02") != "2024-03-10" {
		t.Errorf("Date: got %q, want 2024-03-10", tx.Date.Format("2006-01-02"))
	}
	if tx.ID == 0 {
		t.Error("expected non-zero ID after create")
	}
}

// Req 5.2: Both income and expense types are accepted.
func TestCreate_BothTypes_Accepted(t *testing.T) {
	for _, typ := range []string{"income", "expense"} {
		t.Run("type="+typ, func(t *testing.T) {
			txRepo := newMockTransactionRepo()
			catRepo := newMockCatRepo()
			svc := newTransactionService(txRepo, catRepo)

			cat := seedCat(catRepo, 1, "Test", typ)
			tx, err := svc.Create(1, dto.TransactionRequest{
				CategoryID: cat.ID,
				Amount:     100,
				Type:       typ,
				Date:       "2024-01-01",
			})
			if err != nil {
				t.Fatalf("type %q rejected: %v", typ, err)
			}
			if tx.Type != typ {
				t.Errorf("type mismatch: got %q, want %q", tx.Type, typ)
			}
		})
	}
}

// Req 5.3: Invalid date format returns error.
func TestCreate_InvalidDateFormat_ReturnsError(t *testing.T) {
	invalidDates := []string{"01-15-2024", "2024/01/15", "January 15 2024", "not-a-date"}
	for _, d := range invalidDates {
		t.Run("date="+d, func(t *testing.T) {
			txRepo := newMockTransactionRepo()
			catRepo := newMockCatRepo()
			svc := newTransactionService(txRepo, catRepo)

			cat := seedCat(catRepo, 1, "Food", "expense")
			_, err := svc.Create(1, dto.TransactionRequest{
				CategoryID: cat.ID,
				Amount:     100,
				Type:       "expense",
				Date:       d,
			})
			if err == nil {
				t.Fatalf("expected error for invalid date %q, got nil", d)
			}
		})
	}
}

// Req 5.3: Valid ISO 8601 date (YYYY-MM-DD) is accepted.
func TestCreate_ValidISO8601Date_Accepted(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	cat := seedCat(catRepo, 1, "Salary", "income")
	tx, err := svc.Create(1, dto.TransactionRequest{
		CategoryID: cat.ID,
		Amount:     500,
		Type:       "income",
		Date:       "2024-12-31",
	})
	if err != nil {
		t.Fatalf("unexpected error for valid date: %v", err)
	}
	if tx.Date.Format("2006-01-02") != "2024-12-31" {
		t.Errorf("date mismatch: got %q", tx.Date.Format("2006-01-02"))
	}
}

// Req 5.4: category_id not owned by user returns error.
func TestCreate_CategoryNotOwnedByUser_ReturnsError(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	// Category belongs to user 2
	cat := seedCat(catRepo, 2, "Food", "expense")

	// User 1 tries to use user 2's category
	_, err := svc.Create(1, dto.TransactionRequest{
		CategoryID: cat.ID,
		Amount:     100,
		Type:       "expense",
		Date:       "2024-01-15",
	})
	if err == nil {
		t.Fatal("expected error when using another user's category, got nil")
	}
}

// Req 5.4: Non-existent category_id returns error.
func TestCreate_NonExistentCategory_ReturnsError(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	_, err := svc.Create(1, dto.TransactionRequest{
		CategoryID: 9999,
		Amount:     100,
		Type:       "expense",
		Date:       "2024-01-15",
	})
	if err == nil {
		t.Fatal("expected error for non-existent category, got nil")
	}
}

// Repo error on create is propagated.
func TestCreate_RepoError_Propagated(t *testing.T) {
	txRepo := newMockTransactionRepo()
	txRepo.forceCreateErr = errors.New("db error")
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	cat := seedCat(catRepo, 1, "Food", "expense")
	_, err := svc.Create(1, dto.TransactionRequest{
		CategoryID: cat.ID,
		Amount:     100,
		Type:       "expense",
		Date:       "2024-01-15",
	})
	if err == nil {
		t.Fatal("expected repo error to propagate, got nil")
	}
}

// ─── Update (Req 5.5, 5.7) ───────────────────────────────────────────────────

// Req 5.5: Update succeeds when transaction is owned by user.
func TestUpdateTransaction_ValidOwnership_Succeeds(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	cat := seedCat(catRepo, 1, "Food", "expense")
	tx := seedTx(t, svc, 1, cat.ID, 100, "expense", "2024-01-15", "original")

	updated, err := svc.Update(1, tx.ID, dto.TransactionRequest{
		CategoryID:  cat.ID,
		Amount:      250,
		Type:        "expense",
		Date:        "2024-02-20",
		Description: "updated desc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Amount != 250 {
		t.Errorf("Amount: got %.2f, want 250", updated.Amount)
	}
	if updated.Date.Format("2006-01-02") != "2024-02-20" {
		t.Errorf("Date: got %q, want 2024-02-20", updated.Date.Format("2006-01-02"))
	}
}

// Req 5.5: Update with a category not owned by user returns error.
func TestUpdate_CategoryNotOwnedByUser_ReturnsError(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	cat1 := seedCat(catRepo, 1, "Food", "expense")
	cat2 := seedCat(catRepo, 2, "Other", "expense") // belongs to user 2
	tx := seedTx(t, svc, 1, cat1.ID, 100, "expense", "2024-01-15", "")

	_, err := svc.Update(1, tx.ID, dto.TransactionRequest{
		CategoryID: cat2.ID, // user 2's category
		Amount:     100,
		Type:       "expense",
		Date:       "2024-01-15",
	})
	if err == nil {
		t.Fatal("expected error when updating with another user's category, got nil")
	}
}

// Req 5.7: Update on transaction not owned by user returns error.
func TestUpdateTransaction_WrongOwner_ReturnsError(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	cat1 := seedCat(catRepo, 1, "Food", "expense")
	cat2 := seedCat(catRepo, 2, "Food", "expense")
	tx := seedTx(t, svc, 1, cat1.ID, 100, "expense", "2024-01-15", "")

	// user 2 tries to update user 1's transaction
	_, err := svc.Update(2, tx.ID, dto.TransactionRequest{
		CategoryID: cat2.ID,
		Amount:     999,
		Type:       "expense",
		Date:       "2024-01-15",
	})
	if err == nil {
		t.Fatal("expected error when updating another user's transaction, got nil")
	}
}

// Req 5.7: Update on non-existent transaction returns error.
func TestUpdate_NonExistent_ReturnsError(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	cat := seedCat(catRepo, 1, "Food", "expense")
	_, err := svc.Update(1, 9999, dto.TransactionRequest{
		CategoryID: cat.ID,
		Amount:     100,
		Type:       "expense",
		Date:       "2024-01-15",
	})
	if err == nil {
		t.Fatal("expected error for non-existent transaction, got nil")
	}
}

// Req 5.5: Update with invalid date format returns error.
func TestUpdate_InvalidDate_ReturnsError(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	cat := seedCat(catRepo, 1, "Food", "expense")
	tx := seedTx(t, svc, 1, cat.ID, 100, "expense", "2024-01-15", "")

	_, err := svc.Update(1, tx.ID, dto.TransactionRequest{
		CategoryID: cat.ID,
		Amount:     100,
		Type:       "expense",
		Date:       "15/01/2024",
	})
	if err == nil {
		t.Fatal("expected error for invalid date format, got nil")
	}
}

// ─── Delete (Req 5.6, 5.7) ───────────────────────────────────────────────────

// Req 5.6: Delete succeeds when transaction is owned by user.
func TestDeleteTransaction_ValidOwnership_Succeeds(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	cat := seedCat(catRepo, 1, "Food", "expense")
	tx := seedTx(t, svc, 1, cat.ID, 100, "expense", "2024-01-15", "")

	if err := svc.Delete(1, tx.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's gone
	txs, _, _ := svc.GetAllByUser(1, dto.TransactionFilter{})
	for _, t2 := range txs {
		if t2.ID == tx.ID {
			t.Error("transaction still exists after delete")
		}
	}
}

// Req 5.6: Delete on transaction not owned by user returns error.
func TestDeleteTransaction_WrongOwner_ReturnsError(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	cat := seedCat(catRepo, 1, "Food", "expense")
	tx := seedTx(t, svc, 1, cat.ID, 100, "expense", "2024-01-15", "")

	// user 2 tries to delete user 1's transaction
	err := svc.Delete(2, tx.ID)
	if err == nil {
		t.Fatal("expected error when deleting another user's transaction, got nil")
	}
}

// Req 5.7: Delete on non-existent transaction returns error.
func TestDeleteTransaction_NonExistent_ReturnsError(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	err := svc.Delete(1, 9999)
	if err == nil {
		t.Fatal("expected error for non-existent transaction, got nil")
	}
}

// ─── Filters (Req 5.8) ───────────────────────────────────────────────────────

// Req 5.8: Filter by type returns only matching transactions.
func TestGetAllByUser_FilterByType(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	catIncome := seedCat(catRepo, 1, "Salary", "income")
	catExpense := seedCat(catRepo, 1, "Food", "expense")

	seedTx(t, svc, 1, catIncome.ID, 1000, "income", "2024-01-10", "salary")
	seedTx(t, svc, 1, catExpense.ID, 200, "expense", "2024-01-15", "food")
	seedTx(t, svc, 1, catExpense.ID, 50, "expense", "2024-01-20", "coffee")

	txs, total, err := svc.GetAllByUser(1, dto.TransactionFilter{Type: "expense"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 expense transactions, got %d", total)
	}
	for _, tx := range txs {
		if tx.Type != "expense" {
			t.Errorf("expected expense, got %q", tx.Type)
		}
	}
}

// Req 5.8: Filter by category_id returns only matching transactions.
func TestGetAllByUser_FilterByCategoryID(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	cat1 := seedCat(catRepo, 1, "Food", "expense")
	cat2 := seedCat(catRepo, 1, "Transport", "expense")

	seedTx(t, svc, 1, cat1.ID, 100, "expense", "2024-01-10", "food")
	seedTx(t, svc, 1, cat1.ID, 50, "expense", "2024-01-15", "food2")
	seedTx(t, svc, 1, cat2.ID, 30, "expense", "2024-01-20", "bus")

	txs, total, err := svc.GetAllByUser(1, dto.TransactionFilter{CategoryID: cat1.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 transactions for cat1, got %d", total)
	}
	for _, tx := range txs {
		if tx.CategoryID != cat1.ID {
			t.Errorf("expected categoryID %d, got %d", cat1.ID, tx.CategoryID)
		}
	}
}

// Req 5.8: Filter by start_date excludes earlier transactions.
func TestGetAllByUser_FilterByStartDate(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	cat := seedCat(catRepo, 1, "Food", "expense")
	seedTx(t, svc, 1, cat.ID, 100, "expense", "2024-01-05", "early")
	seedTx(t, svc, 1, cat.ID, 200, "expense", "2024-01-15", "mid")
	seedTx(t, svc, 1, cat.ID, 300, "expense", "2024-01-25", "late")

	txs, total, err := svc.GetAllByUser(1, dto.TransactionFilter{StartDate: "2024-01-10"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 transactions after start_date, got %d", total)
	}
	for _, tx := range txs {
		if tx.Date.Before(time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)) {
			t.Errorf("transaction date %s is before start_date", tx.Date.Format("2006-01-02"))
		}
	}
}

// Req 5.8: Filter by end_date excludes later transactions.
func TestGetAllByUser_FilterByEndDate(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	cat := seedCat(catRepo, 1, "Food", "expense")
	seedTx(t, svc, 1, cat.ID, 100, "expense", "2024-01-05", "early")
	seedTx(t, svc, 1, cat.ID, 200, "expense", "2024-01-15", "mid")
	seedTx(t, svc, 1, cat.ID, 300, "expense", "2024-01-25", "late")

	txs, total, err := svc.GetAllByUser(1, dto.TransactionFilter{EndDate: "2024-01-20"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 transactions before end_date, got %d", total)
	}
	for _, tx := range txs {
		if tx.Date.After(time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)) {
			t.Errorf("transaction date %s is after end_date", tx.Date.Format("2006-01-02"))
		}
	}
}

// Req 5.8: Filter by month returns only transactions in that month.
func TestGetAllByUser_FilterByMonth(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	cat := seedCat(catRepo, 1, "Food", "expense")
	seedTx(t, svc, 1, cat.ID, 100, "expense", "2024-01-15", "jan")
	seedTx(t, svc, 1, cat.ID, 200, "expense", "2024-02-10", "feb")
	seedTx(t, svc, 1, cat.ID, 300, "expense", "2024-02-20", "feb2")

	txs, total, err := svc.GetAllByUser(1, dto.TransactionFilter{Month: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 transactions in February, got %d", total)
	}
	for _, tx := range txs {
		if int(tx.Date.Month()) != 2 {
			t.Errorf("expected month 2, got %d", tx.Date.Month())
		}
	}
}

// Req 5.8: Filter by year returns only transactions in that year.
func TestGetAllByUser_FilterByYear(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	cat := seedCat(catRepo, 1, "Food", "expense")
	seedTx(t, svc, 1, cat.ID, 100, "expense", "2023-06-15", "2023")
	seedTx(t, svc, 1, cat.ID, 200, "expense", "2024-03-10", "2024a")
	seedTx(t, svc, 1, cat.ID, 300, "expense", "2024-09-20", "2024b")

	txs, total, err := svc.GetAllByUser(1, dto.TransactionFilter{Year: 2024})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 transactions in 2024, got %d", total)
	}
	for _, tx := range txs {
		if tx.Date.Year() != 2024 {
			t.Errorf("expected year 2024, got %d", tx.Date.Year())
		}
	}
}

// Req 5.8: Combined month+year filter.
func TestGetAllByUser_FilterByMonthAndYear(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	cat := seedCat(catRepo, 1, "Food", "expense")
	seedTx(t, svc, 1, cat.ID, 100, "expense", "2024-01-10", "jan2024")
	seedTx(t, svc, 1, cat.ID, 200, "expense", "2024-02-10", "feb2024")
	seedTx(t, svc, 1, cat.ID, 300, "expense", "2023-01-10", "jan2023")

	txs, total, err := svc.GetAllByUser(1, dto.TransactionFilter{Month: 1, Year: 2024})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 transaction for Jan 2024, got %d", total)
	}
	if txs[0].Date.Format("2006-01-02") != "2024-01-10" {
		t.Errorf("unexpected date: %s", txs[0].Date.Format("2006-01-02"))
	}
}

// Req 5.8: Combined type + date range filter.
func TestGetAllByUser_FilterByTypeAndDateRange(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	catIncome := seedCat(catRepo, 1, "Salary", "income")
	catExpense := seedCat(catRepo, 1, "Food", "expense")

	seedTx(t, svc, 1, catIncome.ID, 1000, "income", "2024-01-05", "salary")
	seedTx(t, svc, 1, catExpense.ID, 100, "expense", "2024-01-10", "food")
	seedTx(t, svc, 1, catExpense.ID, 200, "expense", "2024-01-20", "food2")
	seedTx(t, svc, 1, catExpense.ID, 300, "expense", "2024-02-01", "food3")

	txs, total, err := svc.GetAllByUser(1, dto.TransactionFilter{
		Type:      "expense",
		StartDate: "2024-01-08",
		EndDate:   "2024-01-25",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 filtered transactions, got %d", total)
	}
	for _, tx := range txs {
		if tx.Type != "expense" {
			t.Errorf("expected expense, got %q", tx.Type)
		}
	}
}

// ─── ExportCSV (Req 11.1–11.3) ───────────────────────────────────────────────

// Req 11.1: ExportCSV returns only the authenticated user's transactions.
func TestExportCSV_ReturnsOnlyOwnTransactions(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	cat1 := seedCat(catRepo, 1, "Salary", "income")
	cat2 := seedCat(catRepo, 2, "Food", "expense")

	seedTx(t, svc, 1, cat1.ID, 1000, "income", "2024-01-15", "user1 salary")
	seedTx(t, svc, 2, cat2.ID, 200, "expense", "2024-01-15", "user2 food")

	data, err := svc.ExportCSV(1, dto.ExportFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := csv.NewReader(strings.NewReader(string(data)))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV: %v", err)
	}

	// header + 1 data row for user 1
	if len(records) != 2 {
		t.Fatalf("expected 2 rows (header + 1 data), got %d", len(records))
	}
	// Verify the data row doesn't contain user 2's description
	if strings.Contains(records[1][5], "user2") {
		t.Error("CSV contains user 2's transaction")
	}
}

// Req 11.2: ExportCSV output is valid CSV with correct headers.
func TestExportCSV_ValidCSVWithHeaders(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	cat := seedCat(catRepo, 1, "Salary", "income")
	seedTx(t, svc, 1, cat.ID, 500, "income", "2024-03-15", "March salary")

	data, err := svc.ExportCSV(1, dto.ExportFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty CSV data")
	}

	r := csv.NewReader(strings.NewReader(string(data)))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("invalid CSV output: %v", err)
	}
	if len(records) < 1 {
		t.Fatal("expected at least a header row")
	}

	// Validate header columns
	header := records[0]
	expectedHeaders := []string{"ID", "Date", "Type", "Category", "Amount", "Description"}
	if len(header) != len(expectedHeaders) {
		t.Fatalf("expected %d header columns, got %d", len(expectedHeaders), len(header))
	}
	for i, h := range expectedHeaders {
		if header[i] != h {
			t.Errorf("header[%d]: got %q, want %q", i, header[i], h)
		}
	}
}

// Req 11.2: ExportCSV data rows contain correct field values.
func TestExportCSV_DataRowsCorrect(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	cat := seedCat(catRepo, 1, "Groceries", "expense")
	// Manually set category name in the mock repo so it appears in CSV
	catRepo.categories[0].Name = "Groceries"

	tx := seedTx(t, svc, 1, cat.ID, 75.50, "expense", "2024-05-10", "Weekly groceries")

	data, err := svc.ExportCSV(1, dto.ExportFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := csv.NewReader(strings.NewReader(string(data)))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("invalid CSV: %v", err)
	}
	if len(records) < 2 {
		t.Fatal("expected header + at least 1 data row")
	}

	row := records[1]
	if row[0] != fmt.Sprintf("%d", tx.ID) {
		t.Errorf("ID: got %q, want %q", row[0], fmt.Sprintf("%d", tx.ID))
	}
	if row[1] != "2024-05-10" {
		t.Errorf("Date: got %q, want 2024-05-10", row[1])
	}
	if row[2] != "expense" {
		t.Errorf("Type: got %q, want expense", row[2])
	}
	if row[4] != "75.50" {
		t.Errorf("Amount: got %q, want 75.50", row[4])
	}
	if row[5] != "Weekly groceries" {
		t.Errorf("Description: got %q, want 'Weekly groceries'", row[5])
	}
}

// Req 11.3: ExportCSV with date range filter returns only transactions in range.
func TestExportCSV_DateRangeFilter(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	cat := seedCat(catRepo, 1, "Food", "expense")
	seedTx(t, svc, 1, cat.ID, 100, "expense", "2024-01-05", "before range")
	seedTx(t, svc, 1, cat.ID, 200, "expense", "2024-01-15", "in range")
	seedTx(t, svc, 1, cat.ID, 300, "expense", "2024-01-25", "after range")

	data, err := svc.ExportCSV(1, dto.ExportFilter{
		StartDate: "2024-01-10",
		EndDate:   "2024-01-20",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := csv.NewReader(strings.NewReader(string(data)))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("invalid CSV: %v", err)
	}

	// header + 1 data row (only "in range")
	if len(records) != 2 {
		t.Fatalf("expected 2 rows (header + 1 in-range), got %d", len(records))
	}
	if records[1][1] != "2024-01-15" {
		t.Errorf("expected date 2024-01-15, got %q", records[1][1])
	}
}

// Req 11.1: ExportCSV for user with no transactions returns only header.
func TestExportCSV_NoTransactions_ReturnsHeaderOnly(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	data, err := svc.ExportCSV(1, dto.ExportFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := csv.NewReader(strings.NewReader(string(data)))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("invalid CSV: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected only header row, got %d rows", len(records))
	}
}

// Req 11.3: ExportCSV with only start_date filter.
func TestExportCSV_StartDateOnly_Filter(t *testing.T) {
	txRepo := newMockTransactionRepo()
	catRepo := newMockCatRepo()
	svc := newTransactionService(txRepo, catRepo)

	cat := seedCat(catRepo, 1, "Food", "expense")
	seedTx(t, svc, 1, cat.ID, 100, "expense", "2024-01-05", "old")
	seedTx(t, svc, 1, cat.ID, 200, "expense", "2024-06-15", "new")

	data, err := svc.ExportCSV(1, dto.ExportFilter{StartDate: "2024-03-01"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := csv.NewReader(strings.NewReader(string(data)))
	records, _ := r.ReadAll()

	// header + 1 row (only "new")
	if len(records) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(records))
	}
	if records[1][1] != "2024-06-15" {
		t.Errorf("expected 2024-06-15, got %q", records[1][1])
	}
}
