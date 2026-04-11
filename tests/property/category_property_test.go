// Package property contains property-based tests for the category feature.
//
// Validates: Requirements 4.1, 4.3, 4.4, 4.5
//
// Run with reduced checks for faster execution:
//
//	go test ./tests/property/ -run "TestProperty9|TestProperty10|TestProperty11|TestProperty12" -rapid.checks=10
package property

import (
	"bytes"
	"encoding/json"
	"fmt"
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
// Category test environment helpers
// ─────────────────────────────────────────────────────────────────────────────

// setupCategoryEnv creates a full test environment with auth + category routes.
func setupCategoryEnv(t *testing.T) *testEnv {
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

// cleanupUser removes a user and all their categories (cascade) by email.
func cleanupUser(db *gorm.DB, email string) {
	var user models.User
	if err := db.Where("email = ?", email).First(&user).Error; err == nil {
		db.Where("user_id = ?", user.ID).Delete(&models.Category{})
	}
	db.Where("email = ?", email).Delete(&models.User{})
}

// loginCounter is used to generate unique IPs for each login call to avoid rate limiting.
var loginCounter int64

// registerAndLogin registers a user (pre-cleaning any stale record first) and
// returns the jwt + csrf cookie values.
// Each call uses a unique source IP to avoid triggering the login rate limiter.
func registerAndLogin(t *testing.T, router *gin.Engine, name, email, password string) (jwtVal, csrfVal string, ok bool) {
	t.Helper()
	return registerAndLoginWithDB(t, router, nil, name, email, password)
}

// registerAndLoginWithDB is like registerAndLogin but pre-cleans any stale DB
// record for the given email before attempting registration.
func registerAndLoginWithDB(t *testing.T, router *gin.Engine, db *gorm.DB, name, email, password string) (jwtVal, csrfVal string, ok bool) {
	t.Helper()

	// Pre-clean stale records so register always succeeds on repeated runs.
	if db != nil {
		cleanupUser(db, email)
	}

	w := doRegister(router, name, email, password)
	if w.Code != http.StatusCreated {
		return "", "", false
	}

	// Use a unique IP per login call to avoid the rate limiter (5 req/min/IP).
	loginCounter++
	ip := fmt.Sprintf("10.%d.%d.%d", (loginCounter/65536)%254+1, (loginCounter/256)%256, loginCounter%256)

	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = ip + ":9999"
	wl := httptest.NewRecorder()
	router.ServeHTTP(wl, req)

	if wl.Code != http.StatusOK {
		return "", "", false
	}

	jwtCookie := extractCookie(wl, "jwt")
	csrfCookie := extractCookie(wl, "csrf_token")
	if jwtCookie == nil || csrfCookie == nil {
		return "", "", false
	}
	return jwtCookie.Value, csrfCookie.Value, true
}

// doGetCategories performs GET /api/categories with the given JWT + CSRF cookies.
func doGetCategories(router *gin.Engine, jwtVal, csrfVal string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/categories", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: jwtVal})
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	req.Header.Set("X-CSRF-Token", csrfVal)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// doCreateCategory performs POST /api/categories.
func doCreateCategory(router *gin.Engine, jwtVal, csrfVal, name, catType string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]string{"name": name, "type": catType})
	req := httptest.NewRequest(http.MethodPost, "/api/categories", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "jwt", Value: jwtVal})
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	req.Header.Set("X-CSRF-Token", csrfVal)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// doUpdateCategory performs PUT /api/categories/:id.
func doUpdateCategory(router *gin.Engine, jwtVal, csrfVal string, id uint, name, catType string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]string{"name": name, "type": catType})
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/categories/%d", id), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "jwt", Value: jwtVal})
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	req.Header.Set("X-CSRF-Token", csrfVal)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// doDeleteCategory performs DELETE /api/categories/:id.
func doDeleteCategory(router *gin.Engine, jwtVal, csrfVal string, id uint) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/categories/%d", id), nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: jwtVal})
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	req.Header.Set("X-CSRF-Token", csrfVal)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// parseCategoryID extracts the category ID from a create/update response body.
func parseCategoryID(body []byte) uint {
	var resp struct {
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(body, &resp)
	return resp.Data.ID
}

// parseCategoryIDs extracts all category IDs from a GET /api/categories response.
func parseCategoryIDs(body []byte) []uint {
	var resp struct {
		Data []struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(body, &resp)
	ids := make([]uint, 0, len(resp.Data))
	for _, d := range resp.Data {
		ids = append(ids, d.ID)
	}
	return ids
}

// validCategoryNameGen generates valid category names (1–50 printable ASCII chars).
func validCategoryNameGen() *rapid.Generator[string] {
	return rapid.StringOfN(
		rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 ")),
		1, 50, -1,
	)
}

// validCategoryTypeGen generates either "income" or "expense".
func validCategoryTypeGen() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{"income", "expense"})
}

// invalidCategoryTypeGen generates strings that are NOT "income" or "expense".
func invalidCategoryTypeGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		candidates := []string{
			"Income", "Expense", "INCOME", "EXPENSE", "income ", " expense",
			"pemasukan", "pengeluaran", "in", "out", "debit", "credit",
			"", "null", "undefined", "0", "true", "Income/Expense",
		}
		if rapid.Bool().Draw(t, "use_candidate") {
			return rapid.SampledFrom(candidates).Draw(t, "candidate")
		}
		s := rapid.StringOfN(
			rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-")),
			0, 20, -1,
		).Draw(t, "random_type")
		if s == "income" || s == "expense" {
			return s + "_invalid"
		}
		return s
	})
}

// uniqueSuffix generates a random 10-char alphanumeric suffix for email addresses.
func uniqueSuffix(t *rapid.T, label string) string {
	return rapid.StringOfN(
		rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyz0123456789")),
		10, 10, -1,
	).Draw(t, label)
}

// ─────────────────────────────────────────────────────────────────────────────
// Property 9: Data isolation between users
// Validates: Requirements 4.1
// ─────────────────────────────────────────────────────────────────────────────

// TestProperty9_DataIsolationBetweenUsers verifies that GET /api/categories only
// returns categories belonging to the authenticated user, never another user's data.
//
// **Validates: Requirements 4.1**
func TestProperty9_DataIsolationBetweenUsers(t *testing.T) {
	env := setupCategoryEnv(t)

	rapid.Check(t, func(rt *rapid.T) {
		emailA := fmt.Sprintf("prop9a%s@test.com", uniqueSuffix(rt, "suffixA"))
		emailB := fmt.Sprintf("prop9b%s@test.com", uniqueSuffix(rt, "suffixB"))
		nameA := validNameGen().Draw(rt, "nameA")
		nameB := validNameGen().Draw(rt, "nameB")
		password := validPasswordGen().Draw(rt, "password")

		t.Cleanup(func() {
			cleanupUser(env.db, emailA)
			cleanupUser(env.db, emailB)
		})

		jwtA, csrfA, okA := registerAndLoginWithDB(t, env.router, env.db, nameA, emailA, password)
		if !okA {
			rt.Fatalf("failed to register/login user A (%s)", emailA)
		}
		jwtB, csrfB, okB := registerAndLoginWithDB(t, env.router, env.db, nameB, emailB, password)
		if !okB {
			rt.Fatalf("failed to register/login user B (%s)", emailB)
		}

		// User A creates a category.
		catNameA := validCategoryNameGen().Draw(rt, "catNameA")
		catTypeA := validCategoryTypeGen().Draw(rt, "catTypeA")
		wCreate := doCreateCategory(env.router, jwtA, csrfA, catNameA, catTypeA)
		if wCreate.Code != http.StatusCreated {
			rt.Fatalf("user A create category failed: %d %s", wCreate.Code, wCreate.Body.String())
		}
		createdID := parseCategoryID(wCreate.Body.Bytes())

		// User B's GET must NOT contain user A's category.
		wGetB := doGetCategories(env.router, jwtB, csrfB)
		if wGetB.Code != http.StatusOK {
			rt.Fatalf("user B GET categories failed: %d %s", wGetB.Code, wGetB.Body.String())
		}
		for _, id := range parseCategoryIDs(wGetB.Body.Bytes()) {
			if id == createdID {
				rt.Fatalf("user B's GET /api/categories returned category ID %d which belongs to user A", createdID)
			}
		}

		// User A's GET must contain the created category.
		wGetA := doGetCategories(env.router, jwtA, csrfA)
		if wGetA.Code != http.StatusOK {
			rt.Fatalf("user A GET categories failed: %d %s", wGetA.Code, wGetA.Body.String())
		}
		found := false
		for _, id := range parseCategoryIDs(wGetA.Body.Bytes()) {
			if id == createdID {
				found = true
				break
			}
		}
		if !found {
			rt.Fatalf("user A's GET /api/categories did not return their own category ID %d", createdID)
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Property 10: Resource ownership enforcement
// Validates: Requirements 4.4
// ─────────────────────────────────────────────────────────────────────────────

// TestProperty10_ResourceOwnershipEnforcement verifies that modifying or deleting
// a category owned by another user is rejected with HTTP 403 or 404.
//
// **Validates: Requirements 4.4**
func TestProperty10_ResourceOwnershipEnforcement(t *testing.T) {
	env := setupCategoryEnv(t)

	rapid.Check(t, func(rt *rapid.T) {
		emailOwner := fmt.Sprintf("prop10o%s@test.com", uniqueSuffix(rt, "suffixO"))
		emailAttacker := fmt.Sprintf("prop10a%s@test.com", uniqueSuffix(rt, "suffixA"))
		nameOwner := validNameGen().Draw(rt, "nameOwner")
		nameAttacker := validNameGen().Draw(rt, "nameAttacker")
		password := validPasswordGen().Draw(rt, "password")

		t.Cleanup(func() {
			cleanupUser(env.db, emailOwner)
			cleanupUser(env.db, emailAttacker)
		})

		jwtOwner, csrfOwner, okOwner := registerAndLoginWithDB(t, env.router, env.db, nameOwner, emailOwner, password)
		if !okOwner {
			rt.Fatalf("failed to register/login owner (%s)", emailOwner)
		}
		jwtAttacker, csrfAttacker, okAttacker := registerAndLoginWithDB(t, env.router, env.db, nameAttacker, emailAttacker, password)
		if !okAttacker {
			rt.Fatalf("failed to register/login attacker (%s)", emailAttacker)
		}

		// Owner creates a category.
		catName := validCategoryNameGen().Draw(rt, "catName")
		catType := validCategoryTypeGen().Draw(rt, "catType")
		wCreate := doCreateCategory(env.router, jwtOwner, csrfOwner, catName, catType)
		if wCreate.Code != http.StatusCreated {
			rt.Fatalf("owner create category failed: %d %s", wCreate.Code, wCreate.Body.String())
		}
		ownerCatID := parseCategoryID(wCreate.Body.Bytes())
		if ownerCatID == 0 {
			rt.Fatalf("could not parse created category ID from: %s", wCreate.Body.String())
		}

		// Attacker tries to UPDATE owner's category — must be 403 or 404.
		wUpdate := doUpdateCategory(env.router, jwtAttacker, csrfAttacker, ownerCatID, "hacked", "expense")
		if wUpdate.Code != http.StatusForbidden && wUpdate.Code != http.StatusNotFound {
			rt.Fatalf("attacker UPDATE owner's category: expected 403 or 404, got %d: %s",
				wUpdate.Code, wUpdate.Body.String())
		}

		// Attacker tries to DELETE owner's category — must be 403 or 404.
		wDelete := doDeleteCategory(env.router, jwtAttacker, csrfAttacker, ownerCatID)
		if wDelete.Code != http.StatusForbidden && wDelete.Code != http.StatusNotFound {
			rt.Fatalf("attacker DELETE owner's category: expected 403 or 404, got %d: %s",
				wDelete.Code, wDelete.Body.String())
		}

		// Owner's category must still exist after the attack attempts.
		wGetOwner := doGetCategories(env.router, jwtOwner, csrfOwner)
		if wGetOwner.Code != http.StatusOK {
			rt.Fatalf("owner GET categories after attack failed: %d %s", wGetOwner.Code, wGetOwner.Body.String())
		}
		stillExists := false
		for _, id := range parseCategoryIDs(wGetOwner.Body.Bytes()) {
			if id == ownerCatID {
				stillExists = true
				break
			}
		}
		if !stillExists {
			rt.Fatalf("owner's category ID %d was deleted or modified by attacker", ownerCatID)
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Property 11: Category type validation rejects all values except income/expense
// Validates: Requirements 4.3
// ─────────────────────────────────────────────────────────────────────────────

// TestProperty11_CategoryTypeValidationRejectsInvalid verifies that any value
// other than "income" or "expense" for the type field is rejected with HTTP 400.
//
// **Validates: Requirements 4.3**
func TestProperty11_CategoryTypeValidationRejectsInvalid(t *testing.T) {
	env := setupCategoryEnv(t)

	rapid.Check(t, func(rt *rapid.T) {
		email := fmt.Sprintf("prop11%s@test.com", uniqueSuffix(rt, "suffix"))
		name := validNameGen().Draw(rt, "name")
		password := validPasswordGen().Draw(rt, "password")

		t.Cleanup(func() { cleanupUser(env.db, email) })

		jwt, csrf, ok := registerAndLoginWithDB(t, env.router, env.db, name, email, password)
		if !ok {
			rt.Fatalf("failed to register/login user (%s)", email)
		}

		invalidType := invalidCategoryTypeGen().Draw(rt, "invalidType")
		catName := validCategoryNameGen().Draw(rt, "catName")

		w := doCreateCategory(env.router, jwt, csrf, catName, invalidType)
		if w.Code != http.StatusBadRequest {
			rt.Fatalf("invalid type %q: expected 400, got %d: %s", invalidType, w.Code, w.Body.String())
		}
	})
}

// TestProperty11b_ValidTypesAlwaysAccepted verifies that "income" and "expense"
// are always accepted as valid category types.
//
// **Validates: Requirements 4.3**
func TestProperty11b_ValidTypesAlwaysAccepted(t *testing.T) {
	env := setupCategoryEnv(t)

	rapid.Check(t, func(rt *rapid.T) {
		email := fmt.Sprintf("prop11b%s@test.com", uniqueSuffix(rt, "suffix"))
		name := validNameGen().Draw(rt, "name")
		password := validPasswordGen().Draw(rt, "password")

		t.Cleanup(func() { cleanupUser(env.db, email) })

		jwt, csrf, ok := registerAndLoginWithDB(t, env.router, env.db, name, email, password)
		if !ok {
			rt.Fatalf("failed to register/login user (%s)", email)
		}

		validType := validCategoryTypeGen().Draw(rt, "validType")
		catName := validCategoryNameGen().Draw(rt, "catName")

		w := doCreateCategory(env.router, jwt, csrf, catName, validType)
		if w.Code != http.StatusCreated {
			rt.Fatalf("valid type %q: expected 201, got %d: %s", validType, w.Code, w.Body.String())
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Property 12: Category uniqueness constraint per user
// Validates: Requirements 4.5
// ─────────────────────────────────────────────────────────────────────────────

// TestProperty12_DuplicateCategoryRejected verifies that creating a category with
// a (name, type) combination that already exists for the same user is rejected
// with HTTP 409 or 400.
//
// **Validates: Requirements 4.5**
func TestProperty12_DuplicateCategoryRejected(t *testing.T) {
	env := setupCategoryEnv(t)

	rapid.Check(t, func(rt *rapid.T) {
		email := fmt.Sprintf("prop12%s@test.com", uniqueSuffix(rt, "suffix"))
		name := validNameGen().Draw(rt, "name")
		password := validPasswordGen().Draw(rt, "password")

		t.Cleanup(func() { cleanupUser(env.db, email) })

		jwt, csrf, ok := registerAndLoginWithDB(t, env.router, env.db, name, email, password)
		if !ok {
			rt.Fatalf("failed to register/login user (%s)", email)
		}

		catName := validCategoryNameGen().Draw(rt, "catName")
		catType := validCategoryTypeGen().Draw(rt, "catType")

		// First creation — must succeed.
		w1 := doCreateCategory(env.router, jwt, csrf, catName, catType)
		if w1.Code != http.StatusCreated {
			rt.Fatalf("first create (%q, %q): expected 201, got %d: %s", catName, catType, w1.Code, w1.Body.String())
		}

		// Second creation with same (name, type) — must be rejected.
		w2 := doCreateCategory(env.router, jwt, csrf, catName, catType)
		if w2.Code != http.StatusConflict && w2.Code != http.StatusBadRequest {
			rt.Fatalf("duplicate (%q, %q): expected 409 or 400, got %d: %s",
				catName, catType, w2.Code, w2.Body.String())
		}
	})
}

// TestProperty12b_SameNameDifferentTypeAllowed verifies that the same category
// name is allowed for different types (income vs expense) for the same user.
//
// **Validates: Requirements 4.5**
func TestProperty12b_SameNameDifferentTypeAllowed(t *testing.T) {
	env := setupCategoryEnv(t)

	rapid.Check(t, func(rt *rapid.T) {
		email := fmt.Sprintf("prop12b%s@test.com", uniqueSuffix(rt, "suffix"))
		name := validNameGen().Draw(rt, "name")
		password := validPasswordGen().Draw(rt, "password")

		t.Cleanup(func() { cleanupUser(env.db, email) })

		jwt, csrf, ok := registerAndLoginWithDB(t, env.router, env.db, name, email, password)
		if !ok {
			rt.Fatalf("failed to register/login user (%s)", email)
		}

		catName := validCategoryNameGen().Draw(rt, "catName")

		// Create as "income".
		w1 := doCreateCategory(env.router, jwt, csrf, catName, "income")
		if w1.Code != http.StatusCreated {
			rt.Fatalf("create income (%q): expected 201, got %d: %s", catName, w1.Code, w1.Body.String())
		}

		// Create same name as "expense" — must succeed (different type).
		w2 := doCreateCategory(env.router, jwt, csrf, catName, "expense")
		if w2.Code != http.StatusCreated {
			rt.Fatalf("create expense with same name (%q): expected 201, got %d: %s", catName, w2.Code, w2.Body.String())
		}
	})
}
