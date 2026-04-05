// Package property contains property-based tests for the budget feature.
//
// Validates: Requirements 6.1, 6.2, 6.4, 6.5, 6.6
//
// Run with reduced checks for faster execution:
//
//	go test ./tests/property/ -run "TestProperty15|TestProperty16|TestProperty17" -rapid.checks=10
package property

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"finance-tracker/config"
	"finance-tracker/controllers"
	"finance-tracker/models"
	"finance-tracker/repositories"
	"finance-tracker/routes"
	"finance-tracker/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"pgregory.net/rapid"
)

// ─────────────────────────────────────────────────────────────────────────────
// Budget test environment helpers
// ─────────────────────────────────────────────────────────────────────────────

// setupBudgetEnv creates a full test environment with auth + category + transaction + budget routes.
func setupBudgetEnv(t *testing.T) *testEnv {
	t.Helper()

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		_ = loadDotEnv("../../.env")
		dbURL = os.Getenv("DB_URL")
	}
	if dbURL == "" {
		t.Skip("DB_URL not set — skipping property tests that require a database")
		return nil
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "test-secret-key-for-property-tests"
	}

	db, err := config.ConnectDatabase(dbURL,
		&models.User{}, &models.Category{}, &models.Transaction{}, &models.Budget{},
	)
	if err != nil {
		t.Skipf("Cannot connect to database: %v — skipping property tests", err)
		return nil
	}

	gin.SetMode(gin.TestMode)

	userRepo := repositories.NewUserRepository(db)
	categoryRepo := repositories.NewCategoryRepository(db)
	transactionRepo := repositories.NewTransactionRepository(db)
	budgetRepo := repositories.NewBudgetRepository(db)

	authSvc := services.NewAuthService(userRepo, jwtSecret)
	categorySvc := services.NewCategoryService(categoryRepo)
	transactionSvc := services.NewTransactionService(transactionRepo, categoryRepo)
	budgetSvc := services.NewBudgetService(budgetRepo, transactionRepo, categoryRepo)
	dashboardSvc := services.NewDashboardService(transactionRepo, budgetRepo)

	authCtrl := controllers.NewAuthController(authSvc)
	categoryCtrl := controllers.NewCategoryController(categorySvc)
	transactionCtrl := controllers.NewTransactionController(transactionSvc)
	budgetCtrl := controllers.NewBudgetController(budgetSvc)
	dashboardCtrl := controllers.NewDashboardController(dashboardSvc)

	cfg := &config.Config{
		JWTSecret:      jwtSecret,
		AllowedOrigins: []string{"http://localhost:5173"},
		Port:           "8080",
		Env:            "test",
	}

	router := routes.SetupRouter(cfg, authCtrl, categoryCtrl, transactionCtrl, budgetCtrl, dashboardCtrl)
	return &testEnv{router: router, db: db}
}

// cleanupBudgets removes all budgets for a user by email.
func cleanupBudgets(t *testing.T, db *gorm.DB, email string) {
	t.Helper()
	var userID uint
	row := db.Raw("SELECT id FROM users WHERE email = ?", email).Row()
	if err := row.Scan(&userID); err != nil || userID == 0 {
		return
	}
	db.Exec("DELETE FROM budgets WHERE user_id = ?", userID)
	db.Exec("DELETE FROM transactions WHERE user_id = ?", userID)
}

// budgetResponse mirrors the BudgetResponse DTO fields returned by the API.
type budgetResponse struct {
	ID          uint    `json:"id"`
	CategoryID  uint    `json:"category_id"`
	Category    string  `json:"category_name"`
	LimitAmount float64 `json:"limit_amount"`
	UsedAmount  float64 `json:"used_amount"`
	Percentage  float64 `json:"percentage"`
	Warning     bool    `json:"warning"`
	Exceeded    bool    `json:"exceeded"`
	Month       int     `json:"month"`
	Year        int     `json:"year"`
}

// doCreateBudget performs POST /api/budgets.
func doCreateBudget(router *gin.Engine, jwtVal, csrfVal string, categoryID uint, limitAmount float64, month, year int) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]interface{}{
		"category_id":  categoryID,
		"limit_amount": limitAmount,
		"month":        month,
		"year":         year,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/budgets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "jwt", Value: jwtVal})
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	req.Header.Set("X-CSRF-Token", csrfVal)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// doGetBudgets performs GET /api/budgets with optional month/year query params.
func doGetBudgets(router *gin.Engine, jwtVal, csrfVal string, month, year int) *httptest.ResponseRecorder {
	url := fmt.Sprintf("/api/budgets?month=%d&year=%d", month, year)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: jwtVal})
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	req.Header.Set("X-CSRF-Token", csrfVal)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// parseBudgetList extracts the list of budgets from a GET /api/budgets response.
func parseBudgetList(body []byte) []budgetResponse {
	var resp struct {
		Data []budgetResponse `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil
	}
	return resp.Data
}

// parseBudgetID extracts the budget ID from a create response body.
func parseBudgetID(body []byte) uint {
	var resp struct {
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(body, &resp)
	return resp.Data.ID
}

// validLimitAmountGen generates positive float64 limit amounts (1.00 – 999999.99).
func validLimitAmountGen() *rapid.Generator[float64] {
	return rapid.Custom(func(t *rapid.T) float64 {
		cents := rapid.IntRange(100, 99999999).Draw(t, "limit_cents")
		return float64(cents) / 100.0
	})
}

// validMonthGen generates a valid month (1–12).
func validMonthGen() *rapid.Generator[int] {
	return rapid.IntRange(1, 12)
}

// validYearGen generates a valid year (2020–2030).
func validYearGen() *rapid.Generator[int] {
	return rapid.IntRange(2020, 2030)
}

// setupBudgetUser registers a user, logs in, and creates one expense category.
// Returns jwt, csrf, expenseCatID.
func setupBudgetUser(t *testing.T, router *gin.Engine, email, name, password string) (jwtVal, csrfVal string, expenseCatID uint, ok bool) {
	t.Helper()

	jwtVal, csrfVal, ok = registerAndLogin(t, router, name, email, password)
	if !ok {
		return "", "", 0, false
	}

	w := doCreateCategory(router, jwtVal, csrfVal, "Expense Cat", "expense")
	if w.Code != http.StatusCreated {
		return "", "", 0, false
	}
	expenseCatID = parseCategoryID(w.Body.Bytes())
	return jwtVal, csrfVal, expenseCatID, true
}

// ─────────────────────────────────────────────────────────────────────────────
// Property 15: Kalkulasi persentase budget selalu akurat — (used/limit) × 100
// Validates: Requirements 6.1, 6.2
// ─────────────────────────────────────────────────────────────────────────────

// TestProperty15_BudgetPercentageCalculationAccurate verifies that for any budget
// with a given limit_amount and any number of expense transactions in the same
// category/month/year, the percentage returned by GET /api/budgets equals
// (used_amount / limit_amount) × 100, rounded to 2 decimal places.
//
// **Validates: Requirements 6.1, 6.2**
func TestProperty15_BudgetPercentageCalculationAccurate(t *testing.T) {
	env := setupBudgetEnv(t)

	rapid.Check(t, func(rt *rapid.T) {
		email := fmt.Sprintf("prop15%s@test.com", uniqueSuffix(rt, "suffix"))
		name := validNameGen().Draw(rt, "name")
		password := validPasswordGen().Draw(rt, "password")

		t.Cleanup(func() {
			cleanupBudgets(t, env.db, email)
			cleanupUser(env.db, email)
		})

		jwtVal, csrfVal, expenseCatID, ok := setupBudgetUser(t, env.router, email, name, password)
		if !ok {
			rt.Fatalf("failed to set up user and category for %s", email)
		}

		limitAmount := validLimitAmountGen().Draw(rt, "limitAmount")
		month := validMonthGen().Draw(rt, "month")
		year := validYearGen().Draw(rt, "year")

		// Create the budget.
		wBudget := doCreateBudget(env.router, jwtVal, csrfVal, expenseCatID, limitAmount, month, year)
		if wBudget.Code != http.StatusCreated {
			rt.Fatalf("create budget failed: %d %s", wBudget.Code, wBudget.Body.String())
		}

		// Create 0–3 expense transactions in the same category/month/year.
		numTx := rapid.IntRange(0, 3).Draw(rt, "numTx")
		var totalUsed float64
		for i := 0; i < numTx; i++ {
			amount := validAmountGen().Draw(rt, fmt.Sprintf("txAmount_%d", i))
			date := fmt.Sprintf("%04d-%02d-%02d", year, month, rapid.IntRange(1, 28).Draw(rt, fmt.Sprintf("day_%d", i)))
			w := doCreateTransaction(env.router, jwtVal, csrfVal, expenseCatID, amount, "expense", date, "test")
			if w.Code != http.StatusCreated {
				rt.Fatalf("create transaction %d failed: %d %s", i, w.Code, w.Body.String())
			}
			totalUsed += amount
		}

		// GET budgets and find the one we created.
		wGet := doGetBudgets(env.router, jwtVal, csrfVal, month, year)
		if wGet.Code != http.StatusOK {
			rt.Fatalf("GET budgets failed: %d %s", wGet.Code, wGet.Body.String())
		}

		budgets := parseBudgetList(wGet.Body.Bytes())
		var found *budgetResponse
		for i := range budgets {
			if budgets[i].CategoryID == expenseCatID && budgets[i].Month == month && budgets[i].Year == year {
				found = &budgets[i]
				break
			}
		}
		if found == nil {
			rt.Fatalf("budget for category %d month %d year %d not found in GET response", expenseCatID, month, year)
		}

		// Verify used_amount matches total transactions (within floating-point tolerance).
		const epsilon = 0.01
		if diff := found.UsedAmount - totalUsed; diff > epsilon || diff < -epsilon {
			rt.Fatalf("used_amount mismatch: expected %.2f, got %.2f", totalUsed, found.UsedAmount)
		}

		// Verify percentage = (used / limit) × 100, rounded to 2 decimal places.
		expectedPct := 0.0
		if limitAmount > 0 {
			expectedPct = math.Round((totalUsed/limitAmount)*10000) / 100
		}
		if diff := found.Percentage - expectedPct; diff > epsilon || diff < -epsilon {
			rt.Fatalf("percentage mismatch: expected %.2f (used=%.2f, limit=%.2f), got %.2f",
				expectedPct, totalUsed, limitAmount, found.Percentage)
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Property 16: Flag warning dan exceeded mengikuti threshold yang benar
// Validates: Requirements 6.5, 6.6
// ─────────────────────────────────────────────────────────────────────────────

// TestProperty16_BudgetWarningAndExceededFlags verifies that:
//   - warning is true if and only if percentage >= 80
//   - exceeded is true if and only if percentage >= 100
//
// **Validates: Requirements 6.5, 6.6**
func TestProperty16_BudgetWarningAndExceededFlags(t *testing.T) {
	env := setupBudgetEnv(t)

	rapid.Check(t, func(rt *rapid.T) {
		email := fmt.Sprintf("prop16%s@test.com", uniqueSuffix(rt, "suffix"))
		name := validNameGen().Draw(rt, "name")
		password := validPasswordGen().Draw(rt, "password")

		t.Cleanup(func() {
			cleanupBudgets(t, env.db, email)
			cleanupUser(env.db, email)
		})

		jwtVal, csrfVal, expenseCatID, ok := setupBudgetUser(t, env.router, email, name, password)
		if !ok {
			rt.Fatalf("failed to set up user and category for %s", email)
		}

		// Use a fixed limit of 100.00 so we can precisely control the percentage
		// by choosing the transaction amount directly.
		const limitAmount = 100.0
		month := validMonthGen().Draw(rt, "month")
		year := validYearGen().Draw(rt, "year")

		wBudget := doCreateBudget(env.router, jwtVal, csrfVal, expenseCatID, limitAmount, month, year)
		if wBudget.Code != http.StatusCreated {
			rt.Fatalf("create budget failed: %d %s", wBudget.Code, wBudget.Body.String())
		}

		// Draw a usage percentage in [0, 150] to cover below-warning, warning, and exceeded zones.
		usagePct := rapid.IntRange(0, 150).Draw(rt, "usagePct")
		usedAmount := float64(usagePct) // since limit=100, used=pct directly

		date := fmt.Sprintf("%04d-%02d-01", year, month)

		if usedAmount > 0 {
			w := doCreateTransaction(env.router, jwtVal, csrfVal, expenseCatID, usedAmount, "expense", date, "test")
			if w.Code != http.StatusCreated {
				rt.Fatalf("create transaction failed: %d %s", w.Code, w.Body.String())
			}
		}

		wGet := doGetBudgets(env.router, jwtVal, csrfVal, month, year)
		if wGet.Code != http.StatusOK {
			rt.Fatalf("GET budgets failed: %d %s", wGet.Code, wGet.Body.String())
		}

		budgets := parseBudgetList(wGet.Body.Bytes())
		var found *budgetResponse
		for i := range budgets {
			if budgets[i].CategoryID == expenseCatID && budgets[i].Month == month && budgets[i].Year == year {
				found = &budgets[i]
				break
			}
		}
		if found == nil {
			rt.Fatalf("budget not found in GET response")
		}

		pct := found.Percentage

		// warning must be true iff percentage >= 80.
		expectedWarning := pct >= 80.0
		if found.Warning != expectedWarning {
			rt.Fatalf("warning flag wrong: percentage=%.2f, expected warning=%v, got warning=%v",
				pct, expectedWarning, found.Warning)
		}

		// exceeded must be true iff percentage >= 100.
		expectedExceeded := pct >= 100.0
		if found.Exceeded != expectedExceeded {
			rt.Fatalf("exceeded flag wrong: percentage=%.2f, expected exceeded=%v, got exceeded=%v",
				pct, expectedExceeded, found.Exceeded)
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Property 17: Uniqueness constraint budget per kombinasi user/kategori/bulan/tahun
// Validates: Requirements 6.4
// ─────────────────────────────────────────────────────────────────────────────

// TestProperty17_BudgetUniquenessConstraint verifies that creating a second budget
// for the same (user_id, category_id, month, year) combination is always rejected,
// while a different combination (different month or year) is always accepted.
//
// **Validates: Requirements 6.4**
func TestProperty17_BudgetUniquenessConstraint(t *testing.T) {
	env := setupBudgetEnv(t)

	rapid.Check(t, func(rt *rapid.T) {
		email := fmt.Sprintf("prop17%s@test.com", uniqueSuffix(rt, "suffix"))
		name := validNameGen().Draw(rt, "name")
		password := validPasswordGen().Draw(rt, "password")

		t.Cleanup(func() {
			cleanupBudgets(t, env.db, email)
			cleanupUser(env.db, email)
		})

		jwtVal, csrfVal, expenseCatID, ok := setupBudgetUser(t, env.router, email, name, password)
		if !ok {
			rt.Fatalf("failed to set up user and category for %s", email)
		}

		limitAmount := validLimitAmountGen().Draw(rt, "limitAmount")
		month := validMonthGen().Draw(rt, "month")
		year := validYearGen().Draw(rt, "year")

		// First creation — must succeed.
		w1 := doCreateBudget(env.router, jwtVal, csrfVal, expenseCatID, limitAmount, month, year)
		if w1.Code != http.StatusCreated {
			rt.Fatalf("first budget create failed: %d %s", w1.Code, w1.Body.String())
		}

		// Second creation with same (category_id, month, year) — must be rejected with 409.
		w2 := doCreateBudget(env.router, jwtVal, csrfVal, expenseCatID, limitAmount+1, month, year)
		if w2.Code != http.StatusConflict {
			rt.Fatalf("duplicate budget: expected 409, got %d: %s", w2.Code, w2.Body.String())
		}

		// Creation with a different month — must succeed (different key).
		differentMonth := month%12 + 1 // rotate month: 12 → 1, others → month+1
		w3 := doCreateBudget(env.router, jwtVal, csrfVal, expenseCatID, limitAmount, differentMonth, year)
		if w3.Code != http.StatusCreated {
			rt.Fatalf("budget with different month failed: expected 201, got %d: %s", w3.Code, w3.Body.String())
		}
	})
}
