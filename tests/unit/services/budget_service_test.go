// Package services_test contains unit tests for BudgetService.
// Validates: Requirements 6.1–6.6
package services_test

import (
	"errors"
	"fmt"
	"testing"

	"finance-tracker/dto"
	"finance-tracker/models"
	"finance-tracker/services"
)

// ─── Mock BudgetRepository ────────────────────────────────────────────────────

type mockBudgetRepo struct {
	budgets        []*models.Budget
	nextID         uint
	forceCreateErr error
	forceUpdateErr error
	forceDeleteErr error
	forceFindErr   error
}

func newMockBudgetRepo() *mockBudgetRepo {
	return &mockBudgetRepo{nextID: 1}
}

func (m *mockBudgetRepo) FindAllByUser(userID uint, month, year int) ([]models.Budget, error) {
	if m.forceFindErr != nil {
		return nil, m.forceFindErr
	}
	var result []models.Budget
	for _, b := range m.budgets {
		if b.UserID == userID && b.Month == month && b.Year == year {
			result = append(result, *b)
		}
	}
	return result, nil
}

func (m *mockBudgetRepo) FindByIDAndUser(id, userID uint) (*models.Budget, error) {
	if m.forceFindErr != nil {
		return nil, m.forceFindErr
	}
	for _, b := range m.budgets {
		if b.ID == id && b.UserID == userID {
			cp := *b
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockBudgetRepo) FindByUserCategoryMonthYear(userID, categoryID uint, month, year int) (*models.Budget, error) {
	if m.forceFindErr != nil {
		return nil, m.forceFindErr
	}
	for _, b := range m.budgets {
		if b.UserID == userID && b.CategoryID == categoryID && b.Month == month && b.Year == year {
			cp := *b
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockBudgetRepo) Create(budget *models.Budget) error {
	if m.forceCreateErr != nil {
		return m.forceCreateErr
	}
	budget.ID = m.nextID
	m.nextID++
	cp := *budget
	m.budgets = append(m.budgets, &cp)
	return nil
}

func (m *mockBudgetRepo) Update(budget *models.Budget) error {
	if m.forceUpdateErr != nil {
		return m.forceUpdateErr
	}
	for i, b := range m.budgets {
		if b.ID == budget.ID {
			cp := *budget
			m.budgets[i] = &cp
			return nil
		}
	}
	return errors.New("not found")
}

func (m *mockBudgetRepo) Delete(id, userID uint) error {
	if m.forceDeleteErr != nil {
		return m.forceDeleteErr
	}
	for i, b := range m.budgets {
		if b.ID == id && b.UserID == userID {
			m.budgets = append(m.budgets[:i], m.budgets[i+1:]...)
			return nil
		}
	}
	return errors.New("not found")
}

// ─── Mock TransactionRepository (partial — only SumByCategory needed) ─────────

type mockTransactionRepoForBudget struct {
	sumByCategory map[string]float64 // key: "userID:catID:month:year"
	forceSumErr   error
}

func newMockTxRepo() *mockTransactionRepoForBudget {
	return &mockTransactionRepoForBudget{sumByCategory: make(map[string]float64)}
}

func (m *mockTransactionRepoForBudget) setSum(userID, catID uint, month, year int, amount float64) {
	key := sumKey(userID, catID, month, year)
	m.sumByCategory[key] = amount
}

func sumKey(userID, catID uint, month, year int) string {
	return fmt.Sprintf("%d:%d:%d:%d", userID, catID, month, year)
}

func (m *mockTransactionRepoForBudget) SumByCategory(userID uint, categoryID uint, month, year int) (float64, error) {
	if m.forceSumErr != nil {
		return 0, m.forceSumErr
	}
	return m.sumByCategory[sumKey(userID, categoryID, month, year)], nil
}

// Unused interface methods — satisfy the interface.
func (m *mockTransactionRepoForBudget) FindAllByUser(userID uint, filter dto.TransactionFilter) ([]models.Transaction, int64, error) {
	return nil, 0, nil
}
func (m *mockTransactionRepoForBudget) FindByIDAndUser(id, userID uint) (*models.Transaction, error) {
	return nil, nil
}
func (m *mockTransactionRepoForBudget) Create(tx *models.Transaction) error { return nil }
func (m *mockTransactionRepoForBudget) Update(tx *models.Transaction) error { return nil }
func (m *mockTransactionRepoForBudget) Delete(id, userID uint) error        { return nil }
func (m *mockTransactionRepoForBudget) GetMonthlyAggregates(userID uint, months int) ([]dto.MonthlyAggregate, error) {
	return nil, nil
}
func (m *mockTransactionRepoForBudget) GetTopExpenseCategories(userID uint, month, year int, limit int) ([]dto.CategoryExpense, error) {
	return nil, nil
}

// ─── Mock CategoryRepository (reuse fields from category_service_test) ────────

// mockCategoryRepoForBudget is a minimal category repo for budget service tests.
type mockCategoryRepoForBudget struct {
	categories []*models.Category
	nextID     uint
}

func newMockCatRepoForBudget() *mockCategoryRepoForBudget {
	return &mockCategoryRepoForBudget{nextID: 1}
}

func (m *mockCategoryRepoForBudget) addCategory(id, userID uint, name, typ string) {
	m.categories = append(m.categories, &models.Category{ID: id, UserID: userID, Name: name, Type: typ})
}

func (m *mockCategoryRepoForBudget) FindAllByUser(userID uint) ([]models.Category, error) {
	var result []models.Category
	for _, c := range m.categories {
		if c.UserID == userID {
			result = append(result, *c)
		}
	}
	return result, nil
}

func (m *mockCategoryRepoForBudget) FindByIDAndUser(id, userID uint) (*models.Category, error) {
	for _, c := range m.categories {
		if c.ID == id && c.UserID == userID {
			cp := *c
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockCategoryRepoForBudget) Create(category *models.Category) error {
	category.ID = m.nextID
	m.nextID++
	cp := *category
	m.categories = append(m.categories, &cp)
	return nil
}

func (m *mockCategoryRepoForBudget) Update(category *models.Category) error { return nil }
func (m *mockCategoryRepoForBudget) Delete(id, userID uint) error           { return nil }
func (m *mockCategoryRepoForBudget) ExistsByNameTypeUser(name, categoryType string, userID uint) (bool, error) {
	return false, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func newBudgetService(budgetRepo *mockBudgetRepo, txRepo *mockTransactionRepoForBudget, catRepo *mockCategoryRepoForBudget) services.BudgetService {
	return services.NewBudgetService(budgetRepo, txRepo, catRepo)
}

// seedBudget creates a budget via the service and fails the test on error.
func seedBudget(t *testing.T, svc services.BudgetService, userID, catID uint, limit float64, month, year int) *models.Budget {
	t.Helper()
	b, err := svc.Create(userID, dto.BudgetRequest{
		CategoryID:  catID,
		LimitAmount: limit,
		Month:       month,
		Year:        year,
	})
	if err != nil {
		t.Fatalf("seedBudget failed: %v", err)
	}
	return b
}

// ─── GetAllByUser ─────────────────────────────────────────────────────────────

// Req 6.1: GetAllByUser returns budgets with usage percentage for the given month/year.
func TestBudgetGetAllByUser_ReturnsOnlyMatchingMonthYear(t *testing.T) {
	budgetRepo := newMockBudgetRepo()
	txRepo := newMockTxRepo()
	catRepo := newMockCatRepoForBudget()
	catRepo.addCategory(1, 1, "Food", "expense")
	catRepo.addCategory(2, 1, "Transport", "expense")

	svc := newBudgetService(budgetRepo, txRepo, catRepo)

	seedBudget(t, svc, 1, 1, 500.0, 3, 2024)
	seedBudget(t, svc, 1, 2, 200.0, 4, 2024) // different month

	results, err := svc.GetAllByUser(1, 3, 2024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 budget for month=3 year=2024, got %d", len(results))
	}
	if results[0].CategoryID != 1 {
		t.Errorf("expected category 1, got %d", results[0].CategoryID)
	}
}

// Req 6.1: GetAllByUser returns empty slice when no budgets exist for that period.
func TestBudgetGetAllByUser_EmptyForNoBudgets(t *testing.T) {
	svc := newBudgetService(newMockBudgetRepo(), newMockTxRepo(), newMockCatRepoForBudget())

	results, err := svc.GetAllByUser(1, 1, 2024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 budgets, got %d", len(results))
	}
}

// Req 6.1: GetAllByUser populates used_amount from TransactionRepository.SumByCategory.
func TestBudgetGetAllByUser_UsedAmountFromTransactions(t *testing.T) {
	budgetRepo := newMockBudgetRepo()
	txRepo := newMockTxRepo()
	catRepo := newMockCatRepoForBudget()
	catRepo.addCategory(1, 1, "Food", "expense")

	svc := newBudgetService(budgetRepo, txRepo, catRepo)
	seedBudget(t, svc, 1, 1, 1000.0, 5, 2024)

	txRepo.setSum(1, 1, 5, 2024, 350.0)

	results, err := svc.GetAllByUser(1, 5, 2024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].UsedAmount != 350.0 {
		t.Errorf("UsedAmount: got %.2f, want 350.00", results[0].UsedAmount)
	}
}

// Req 6.1: Percentage = (used / limit) * 100, rounded to 2 decimal places.
func TestBudgetGetAllByUser_PercentageCalculation(t *testing.T) {
	tests := []struct {
		name        string
		limit       float64
		used        float64
		wantPct     float64
		wantWarning bool
		wantExceed  bool
	}{
		{"zero usage", 100.0, 0.0, 0.0, false, false},
		{"50 percent", 200.0, 100.0, 50.0, false, false},
		{"79.99 percent — below warning", 100.0, 79.99, 79.99, false, false},
		{"exactly 80 percent — warning", 100.0, 80.0, 80.0, true, false},
		{"85 percent — warning", 100.0, 85.0, 85.0, true, false},
		{"99.99 percent — warning not exceeded", 100.0, 99.99, 99.99, true, false},
		{"exactly 100 percent — exceeded", 100.0, 100.0, 100.0, true, true},
		{"150 percent — exceeded", 100.0, 150.0, 150.0, true, true},
		{"fractional rounding", 300.0, 100.0, 33.33, false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			budgetRepo := newMockBudgetRepo()
			txRepo := newMockTxRepo()
			catRepo := newMockCatRepoForBudget()
			catRepo.addCategory(1, 1, "Food", "expense")

			svc := newBudgetService(budgetRepo, txRepo, catRepo)
			seedBudget(t, svc, 1, 1, tc.limit, 1, 2024)
			txRepo.setSum(1, 1, 1, 2024, tc.used)

			results, err := svc.GetAllByUser(1, 1, 2024)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}
			r := results[0]

			const epsilon = 0.005
			if diff := r.Percentage - tc.wantPct; diff > epsilon || diff < -epsilon {
				t.Errorf("Percentage: got %.4f, want %.4f", r.Percentage, tc.wantPct)
			}
			if r.Warning != tc.wantWarning {
				t.Errorf("Warning: got %v, want %v (pct=%.2f)", r.Warning, tc.wantWarning, r.Percentage)
			}
			if r.Exceeded != tc.wantExceed {
				t.Errorf("Exceeded: got %v, want %v (pct=%.2f)", r.Exceeded, tc.wantExceed, r.Percentage)
			}
		})
	}
}

// ─── Create ───────────────────────────────────────────────────────────────────

// Req 6.2: Create with valid fields saves budget correctly.
func TestBudgetCreate_ValidInput(t *testing.T) {
	budgetRepo := newMockBudgetRepo()
	txRepo := newMockTxRepo()
	catRepo := newMockCatRepoForBudget()
	catRepo.addCategory(1, 1, "Food", "expense")

	svc := newBudgetService(budgetRepo, txRepo, catRepo)

	b, err := svc.Create(1, dto.BudgetRequest{
		CategoryID:  1,
		LimitAmount: 500.0,
		Month:       6,
		Year:        2024,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b == nil {
		t.Fatal("expected budget, got nil")
	}
	if b.UserID != 1 {
		t.Errorf("UserID: got %d, want 1", b.UserID)
	}
	if b.CategoryID != 1 {
		t.Errorf("CategoryID: got %d, want 1", b.CategoryID)
	}
	if b.LimitAmount != 500.0 {
		t.Errorf("LimitAmount: got %.2f, want 500.00", b.LimitAmount)
	}
	if b.Month != 6 {
		t.Errorf("Month: got %d, want 6", b.Month)
	}
	if b.Year != 2024 {
		t.Errorf("Year: got %d, want 2024", b.Year)
	}
	if b.ID == 0 {
		t.Error("expected non-zero ID after create")
	}
}

// Req 6.2: Create with category belonging to another user returns error.
func TestBudgetCreate_CategoryNotOwnedByUser_ReturnsError(t *testing.T) {
	budgetRepo := newMockBudgetRepo()
	txRepo := newMockTxRepo()
	catRepo := newMockCatRepoForBudget()
	catRepo.addCategory(1, 2, "Food", "expense") // belongs to user 2

	svc := newBudgetService(budgetRepo, txRepo, catRepo)

	_, err := svc.Create(1, dto.BudgetRequest{
		CategoryID:  1,
		LimitAmount: 100.0,
		Month:       1,
		Year:        2024,
	})
	if err == nil {
		t.Fatal("expected error for category not owned by user, got nil")
	}
}

// Req 6.4: Create duplicate (user_id, category_id, month, year) returns error.
func TestBudgetCreate_DuplicateCombo_ReturnsError(t *testing.T) {
	budgetRepo := newMockBudgetRepo()
	txRepo := newMockTxRepo()
	catRepo := newMockCatRepoForBudget()
	catRepo.addCategory(1, 1, "Food", "expense")

	svc := newBudgetService(budgetRepo, txRepo, catRepo)

	req := dto.BudgetRequest{CategoryID: 1, LimitAmount: 100.0, Month: 3, Year: 2024}
	if _, err := svc.Create(1, req); err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	_, err := svc.Create(1, dto.BudgetRequest{CategoryID: 1, LimitAmount: 200.0, Month: 3, Year: 2024})
	if err == nil {
		t.Fatal("expected error for duplicate budget, got nil")
	}
}

// Req 6.4: Same category + different month is allowed.
func TestBudgetCreate_SameCategoryDifferentMonth_Allowed(t *testing.T) {
	budgetRepo := newMockBudgetRepo()
	txRepo := newMockTxRepo()
	catRepo := newMockCatRepoForBudget()
	catRepo.addCategory(1, 1, "Food", "expense")

	svc := newBudgetService(budgetRepo, txRepo, catRepo)

	if _, err := svc.Create(1, dto.BudgetRequest{CategoryID: 1, LimitAmount: 100.0, Month: 3, Year: 2024}); err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	_, err := svc.Create(1, dto.BudgetRequest{CategoryID: 1, LimitAmount: 100.0, Month: 4, Year: 2024})
	if err != nil {
		t.Fatalf("different month should be allowed, got: %v", err)
	}
}

// Req 6.4: Same category + same month + different year is allowed.
func TestBudgetCreate_SameCategoryDifferentYear_Allowed(t *testing.T) {
	budgetRepo := newMockBudgetRepo()
	txRepo := newMockTxRepo()
	catRepo := newMockCatRepoForBudget()
	catRepo.addCategory(1, 1, "Food", "expense")

	svc := newBudgetService(budgetRepo, txRepo, catRepo)

	if _, err := svc.Create(1, dto.BudgetRequest{CategoryID: 1, LimitAmount: 100.0, Month: 3, Year: 2024}); err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	_, err := svc.Create(1, dto.BudgetRequest{CategoryID: 1, LimitAmount: 100.0, Month: 3, Year: 2025})
	if err != nil {
		t.Fatalf("different year should be allowed, got: %v", err)
	}
}

// Req 6.4: Same combo for different users is allowed.
func TestBudgetCreate_SameCombo_DifferentUsers_Allowed(t *testing.T) {
	budgetRepo := newMockBudgetRepo()
	txRepo := newMockTxRepo()
	catRepo := newMockCatRepoForBudget()
	catRepo.addCategory(1, 1, "Food", "expense")
	catRepo.addCategory(2, 2, "Food", "expense")

	svc := newBudgetService(budgetRepo, txRepo, catRepo)

	if _, err := svc.Create(1, dto.BudgetRequest{CategoryID: 1, LimitAmount: 100.0, Month: 3, Year: 2024}); err != nil {
		t.Fatalf("user 1 create failed: %v", err)
	}
	_, err := svc.Create(2, dto.BudgetRequest{CategoryID: 2, LimitAmount: 100.0, Month: 3, Year: 2024})
	if err != nil {
		t.Fatalf("user 2 should be able to create same combo, got: %v", err)
	}
}

// ─── Warning / Exceeded threshold edge cases (Req 6.5, 6.6) ──────────────────

// Req 6.5: Warning flag is true when usage is exactly 80%.
func TestBudgetWarning_ExactlyEightyPercent(t *testing.T) {
	budgetRepo := newMockBudgetRepo()
	txRepo := newMockTxRepo()
	catRepo := newMockCatRepoForBudget()
	catRepo.addCategory(1, 1, "Food", "expense")

	svc := newBudgetService(budgetRepo, txRepo, catRepo)
	seedBudget(t, svc, 1, 1, 100.0, 1, 2024)
	txRepo.setSum(1, 1, 1, 2024, 80.0)

	results, _ := svc.GetAllByUser(1, 1, 2024)
	if !results[0].Warning {
		t.Error("expected warning=true at exactly 80%")
	}
	if results[0].Exceeded {
		t.Error("expected exceeded=false at 80%")
	}
}

// Req 6.5: Warning flag is false just below 80%.
func TestBudgetWarning_JustBelowEightyPercent(t *testing.T) {
	budgetRepo := newMockBudgetRepo()
	txRepo := newMockTxRepo()
	catRepo := newMockCatRepoForBudget()
	catRepo.addCategory(1, 1, "Food", "expense")

	svc := newBudgetService(budgetRepo, txRepo, catRepo)
	seedBudget(t, svc, 1, 1, 100.0, 1, 2024)
	txRepo.setSum(1, 1, 1, 2024, 79.99)

	results, _ := svc.GetAllByUser(1, 1, 2024)
	if results[0].Warning {
		t.Errorf("expected warning=false at 79.99%%, got true (pct=%.2f)", results[0].Percentage)
	}
}

// Req 6.6: Exceeded flag is true when usage is exactly 100%.
func TestBudgetExceeded_ExactlyOneHundredPercent(t *testing.T) {
	budgetRepo := newMockBudgetRepo()
	txRepo := newMockTxRepo()
	catRepo := newMockCatRepoForBudget()
	catRepo.addCategory(1, 1, "Food", "expense")

	svc := newBudgetService(budgetRepo, txRepo, catRepo)
	seedBudget(t, svc, 1, 1, 100.0, 1, 2024)
	txRepo.setSum(1, 1, 1, 2024, 100.0)

	results, _ := svc.GetAllByUser(1, 1, 2024)
	if !results[0].Warning {
		t.Error("expected warning=true at 100%")
	}
	if !results[0].Exceeded {
		t.Error("expected exceeded=true at exactly 100%")
	}
}

// Req 6.6: Exceeded flag is false just below 100%.
func TestBudgetExceeded_JustBelowOneHundredPercent(t *testing.T) {
	budgetRepo := newMockBudgetRepo()
	txRepo := newMockTxRepo()
	catRepo := newMockCatRepoForBudget()
	catRepo.addCategory(1, 1, "Food", "expense")

	svc := newBudgetService(budgetRepo, txRepo, catRepo)
	seedBudget(t, svc, 1, 1, 100.0, 1, 2024)
	txRepo.setSum(1, 1, 1, 2024, 99.99)

	results, _ := svc.GetAllByUser(1, 1, 2024)
	if !results[0].Warning {
		t.Error("expected warning=true at 99.99%")
	}
	if results[0].Exceeded {
		t.Errorf("expected exceeded=false at 99.99%%, got true (pct=%.2f)", results[0].Percentage)
	}
}

// Req 6.6: Exceeded flag is true above 100%.
func TestBudgetExceeded_AboveOneHundredPercent(t *testing.T) {
	budgetRepo := newMockBudgetRepo()
	txRepo := newMockTxRepo()
	catRepo := newMockCatRepoForBudget()
	catRepo.addCategory(1, 1, "Food", "expense")

	svc := newBudgetService(budgetRepo, txRepo, catRepo)
	seedBudget(t, svc, 1, 1, 100.0, 1, 2024)
	txRepo.setSum(1, 1, 1, 2024, 150.0)

	results, _ := svc.GetAllByUser(1, 1, 2024)
	if !results[0].Exceeded {
		t.Error("expected exceeded=true at 150%")
	}
}

// ─── Update ───────────────────────────────────────────────────────────────────

// Update with valid ownership succeeds and persists new values.
func TestBudgetUpdate_ValidOwnership_Succeeds(t *testing.T) {
	budgetRepo := newMockBudgetRepo()
	txRepo := newMockTxRepo()
	catRepo := newMockCatRepoForBudget()
	catRepo.addCategory(1, 1, "Food", "expense")
	catRepo.addCategory(2, 1, "Transport", "expense")

	svc := newBudgetService(budgetRepo, txRepo, catRepo)
	b := seedBudget(t, svc, 1, 1, 500.0, 3, 2024)

	updated, err := svc.Update(1, b.ID, dto.BudgetRequest{
		CategoryID:  2,
		LimitAmount: 750.0,
		Month:       3,
		Year:        2024,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.LimitAmount != 750.0 {
		t.Errorf("LimitAmount: got %.2f, want 750.00", updated.LimitAmount)
	}
	if updated.CategoryID != 2 {
		t.Errorf("CategoryID: got %d, want 2", updated.CategoryID)
	}
}

// Update with budgetID belonging to another user returns error.
func TestBudgetUpdate_WrongOwner_ReturnsError(t *testing.T) {
	budgetRepo := newMockBudgetRepo()
	txRepo := newMockTxRepo()
	catRepo := newMockCatRepoForBudget()
	catRepo.addCategory(1, 1, "Food", "expense")

	svc := newBudgetService(budgetRepo, txRepo, catRepo)
	b := seedBudget(t, svc, 1, 1, 500.0, 3, 2024)

	_, err := svc.Update(2, b.ID, dto.BudgetRequest{
		CategoryID:  1,
		LimitAmount: 100.0,
		Month:       3,
		Year:        2024,
	})
	if err == nil {
		t.Fatal("expected error when updating another user's budget, got nil")
	}
}

// Update to a duplicate (category_id, month, year) for the same user returns error.
func TestBudgetUpdate_DuplicateCombo_ReturnsError(t *testing.T) {
	budgetRepo := newMockBudgetRepo()
	txRepo := newMockTxRepo()
	catRepo := newMockCatRepoForBudget()
	catRepo.addCategory(1, 1, "Food", "expense")
	catRepo.addCategory(2, 1, "Transport", "expense")

	svc := newBudgetService(budgetRepo, txRepo, catRepo)
	seedBudget(t, svc, 1, 1, 500.0, 3, 2024)
	b2 := seedBudget(t, svc, 1, 2, 300.0, 3, 2024)

	// Try to update b2 to use category 1 / month 3 / year 2024 — conflicts with first budget.
	_, err := svc.Update(1, b2.ID, dto.BudgetRequest{
		CategoryID:  1,
		LimitAmount: 300.0,
		Month:       3,
		Year:        2024,
	})
	if err == nil {
		t.Fatal("expected error for duplicate combo on update, got nil")
	}
}

// Update with same key fields (no combo change) should succeed.
func TestBudgetUpdate_SameCombo_NoChange_Succeeds(t *testing.T) {
	budgetRepo := newMockBudgetRepo()
	txRepo := newMockTxRepo()
	catRepo := newMockCatRepoForBudget()
	catRepo.addCategory(1, 1, "Food", "expense")

	svc := newBudgetService(budgetRepo, txRepo, catRepo)
	b := seedBudget(t, svc, 1, 1, 500.0, 3, 2024)

	_, err := svc.Update(1, b.ID, dto.BudgetRequest{
		CategoryID:  1,
		LimitAmount: 600.0, // only limit changes
		Month:       3,
		Year:        2024,
	})
	if err != nil {
		t.Fatalf("updating limit only should succeed, got: %v", err)
	}
}

// Update with category not owned by user returns error.
func TestBudgetUpdate_CategoryNotOwned_ReturnsError(t *testing.T) {
	budgetRepo := newMockBudgetRepo()
	txRepo := newMockTxRepo()
	catRepo := newMockCatRepoForBudget()
	catRepo.addCategory(1, 1, "Food", "expense")
	catRepo.addCategory(2, 2, "Other", "expense") // belongs to user 2

	svc := newBudgetService(budgetRepo, txRepo, catRepo)
	b := seedBudget(t, svc, 1, 1, 500.0, 3, 2024)

	_, err := svc.Update(1, b.ID, dto.BudgetRequest{
		CategoryID:  2, // user 2's category
		LimitAmount: 500.0,
		Month:       3,
		Year:        2024,
	})
	if err == nil {
		t.Fatal("expected error for category not owned by user, got nil")
	}
}

// ─── Delete ───────────────────────────────────────────────────────────────────

// Delete with valid ownership removes the budget.
func TestBudgetDelete_ValidOwnership_Succeeds(t *testing.T) {
	budgetRepo := newMockBudgetRepo()
	txRepo := newMockTxRepo()
	catRepo := newMockCatRepoForBudget()
	catRepo.addCategory(1, 1, "Food", "expense")

	svc := newBudgetService(budgetRepo, txRepo, catRepo)
	b := seedBudget(t, svc, 1, 1, 500.0, 3, 2024)

	if err := svc.Delete(1, b.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	results, _ := svc.GetAllByUser(1, 3, 2024)
	for _, r := range results {
		if r.ID == b.ID {
			t.Error("budget still present after delete")
		}
	}
}

// Delete with budgetID belonging to another user returns error.
func TestBudgetDelete_WrongOwner_ReturnsError(t *testing.T) {
	budgetRepo := newMockBudgetRepo()
	txRepo := newMockTxRepo()
	catRepo := newMockCatRepoForBudget()
	catRepo.addCategory(1, 1, "Food", "expense")

	svc := newBudgetService(budgetRepo, txRepo, catRepo)
	b := seedBudget(t, svc, 1, 1, 500.0, 3, 2024)

	err := svc.Delete(2, b.ID)
	if err == nil {
		t.Fatal("expected error when deleting another user's budget, got nil")
	}
}

// Delete non-existent budget returns error.
func TestBudgetDelete_NonExistent_ReturnsError(t *testing.T) {
	svc := newBudgetService(newMockBudgetRepo(), newMockTxRepo(), newMockCatRepoForBudget())

	err := svc.Delete(1, 999)
	if err == nil {
		t.Fatal("expected error for non-existent budget, got nil")
	}
}
