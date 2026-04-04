// Package property contains property-based tests for the authentication feature.
//
// Validates: Requirements 1.2, 1.3, 1.4, 2.2, 2.3, 2.4, 2.6, 3.1, 3.2, 3.3, 3.4
//
// Run with reduced checks for faster execution:
//
//	go test ./tests/property/ -rapid.checks=10
package property

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"finance-tracker/config"
	"finance-tracker/controllers"
	"finance-tracker/middleware"
	"finance-tracker/models"
	"finance-tracker/repositories"
	"finance-tracker/routes"
	"finance-tracker/services"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"pgregory.net/rapid"
)

// jwtPattern matches a JWT string: three base64url segments separated by dots.
var jwtPattern = regexp.MustCompile(`[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`)

// TestMain sets rapid.checks to 10 so the suite runs faster.
// Override at the command line with -rapid.checks=N if needed.
func TestMain(m *testing.M) {
	// rapid registers its flags via init(); only override if still at the default of 100.
	if f := flag.Lookup("rapid.checks"); f != nil && f.Value.String() == "100" {
		_ = flag.Set("rapid.checks", "10")
	}
	os.Exit(m.Run())
}

// testEnv holds shared test infrastructure.
type testEnv struct {
	router *gin.Engine
	db     *gorm.DB
}

// setupTestEnv creates a Gin router wired to a real PostgreSQL test DB.
func setupTestEnv(t *testing.T) *testEnv {
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
	authSvc := services.NewAuthService(userRepo, jwtSecret)
	categorySvc := services.NewCategoryService(categoryRepo)
	authCtrl := controllers.NewAuthController(authSvc)
	categoryCtrl := controllers.NewCategoryController(categorySvc)

	cfg := &config.Config{
		JWTSecret:      jwtSecret,
		AllowedOrigins: []string{"http://localhost:5173"},
		Port:           "8080",
		Env:            "test",
	}

	router := routes.SetupRouter(cfg, authCtrl, categoryCtrl)

	// Minimal protected group for Properties 6 and 8.
	protected := router.Group("/api/protected")
	protected.Use(middleware.JWTAuthMiddleware(jwtSecret))
	protected.Use(middleware.CSRFMiddleware())
	protected.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "ok"}) })
	protected.POST("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "ok"}) })
	protected.PUT("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "ok"}) })
	protected.DELETE("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "ok"}) })

	return &testEnv{router: router, db: db}
}

func loadDotEnv(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, val := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
	return nil
}

func cleanupEmail(db *gorm.DB, email string) {
	db.Where("email = ?", email).Delete(&models.User{})
}

func doRegister(router *gin.Engine, name, email, password string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]string{"name": name, "email": email, "password": password})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func doLogin(router *gin.Engine, email, password string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// doLoginWithIP sends a login request appearing to come from a specific IP.
// This avoids the rate limiter when running many rapid iterations.
func doLoginWithIP(router *gin.Engine, email, password, ip string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = ip + ":12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func extractCookie(w *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range w.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func validNameGen() *rapid.Generator[string] {
	return rapid.StringOfN(rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ ")), 2, 50, -1)
}

func validPasswordGen() *rapid.Generator[string] {
	return rapid.StringOfN(rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%")), 8, 30, -1)
}

func uniqueEmailGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		suffix := rapid.StringOfN(rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyz0123456789")), 8, 12, -1).Draw(t, "suffix")
		return fmt.Sprintf("prop%s@test.com", suffix)
	})
}

func uniqueIPGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		a := rapid.IntRange(1, 254).Draw(t, "ip_a")
		b := rapid.IntRange(0, 255).Draw(t, "ip_b")
		c := rapid.IntRange(0, 255).Draw(t, "ip_c")
		return fmt.Sprintf("10.%d.%d.%d", a, b, c)
	})
}

// Validates: Requirements 1.2
// ─────────────────────────────────────────────────────────────────────────────

func TestProperty1_BcryptHashCostAtLeast12(t *testing.T) {
	env := setupTestEnv(t)
	rapid.Check(t, func(rt *rapid.T) {
		name := validNameGen().Draw(rt, "name")
		email := uniqueEmailGen().Draw(rt, "email")
		password := validPasswordGen().Draw(rt, "password")
		t.Cleanup(func() { cleanupEmail(env.db, email) })

		w := doRegister(env.router, name, email, password)
		if w.Code != http.StatusCreated {
			rt.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var user models.User
		if err := env.db.Where("email = ?", email).First(&user).Error; err != nil {
			rt.Fatalf("user not found in DB: %v", err)
		}
		if user.Password == password {
			rt.Fatal("password stored as plaintext — must be hashed")
		}
		cost, err := bcrypt.Cost([]byte(user.Password))
		if err != nil {
			rt.Fatalf("stored value is not a valid bcrypt hash: %v", err)
		}
		if cost < 12 {
			rt.Fatalf("bcrypt cost %d < required minimum 12", cost)
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Property 2: Duplicate email always rejected with HTTP 409
// Validates: Requirements 1.3
// ─────────────────────────────────────────────────────────────────────────────

func TestProperty2_DuplicateEmailRejectedWith409(t *testing.T) {
	env := setupTestEnv(t)
	rapid.Check(t, func(rt *rapid.T) {
		name := validNameGen().Draw(rt, "name")
		email := uniqueEmailGen().Draw(rt, "email")
		password := validPasswordGen().Draw(rt, "password")
		t.Cleanup(func() { cleanupEmail(env.db, email) })

		w1 := doRegister(env.router, name, email, password)
		if w1.Code != http.StatusCreated {
			rt.Fatalf("first registration failed with %d: %s", w1.Code, w1.Body.String())
		}
		w2 := doRegister(env.router, name, email, password)
		if w2.Code != http.StatusConflict {
			rt.Fatalf("duplicate email: expected 409, got %d: %s", w2.Code, w2.Body.String())
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Property 3: Invalid input always rejected with HTTP 400
// Validates: Requirements 1.4
// ─────────────────────────────────────────────────────────────────────────────

func TestProperty3_InvalidInputRejectedWith400(t *testing.T) {
	env := setupTestEnv(t)

	t.Run("invalid_email_format", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			name := validNameGen().Draw(rt, "name")
			password := validPasswordGen().Draw(rt, "password")
			invalidEmail := rapid.StringOfN(
				rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyz0123456789")), 1, 20, -1,
			).Draw(rt, "invalidEmail")
			w := doRegister(env.router, name, invalidEmail, password)
			if w.Code != http.StatusBadRequest {
				rt.Fatalf("invalid email %q: expected 400, got %d: %s", invalidEmail, w.Code, w.Body.String())
			}
		})
	})

	t.Run("password_too_short", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			name := validNameGen().Draw(rt, "name")
			email := uniqueEmailGen().Draw(rt, "email")
			shortPassword := rapid.StringOfN(
				rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyz0123456789")), 1, 7, -1,
			).Draw(rt, "shortPassword")
			w := doRegister(env.router, name, email, shortPassword)
			if w.Code != http.StatusBadRequest {
				rt.Fatalf("short password %q: expected 400, got %d: %s", shortPassword, w.Code, w.Body.String())
			}
		})
	})

	t.Run("name_too_short", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			email := uniqueEmailGen().Draw(rt, "email")
			password := validPasswordGen().Draw(rt, "password")
			shortName := rapid.StringOfN(
				rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyz")), 1, 1, -1,
			).Draw(rt, "shortName")
			w := doRegister(env.router, shortName, email, password)
			if w.Code != http.StatusBadRequest {
				rt.Fatalf("short name %q: expected 400, got %d: %s", shortName, w.Code, w.Body.String())
			}
		})
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Property 4: Successful login produces JWT only via HttpOnly Cookie
// Validates: Requirements 2.2, 2.3, 2.4
// ─────────────────────────────────────────────────────────────────────────────

func TestProperty4_LoginJWTOnlyInHttpOnlyCookie(t *testing.T) {
	env := setupTestEnv(t)
	rapid.Check(t, func(rt *rapid.T) {
		name := validNameGen().Draw(rt, "name")
		email := uniqueEmailGen().Draw(rt, "email")
		password := validPasswordGen().Draw(rt, "password")
		ip := uniqueIPGen().Draw(rt, "ip")
		t.Cleanup(func() { cleanupEmail(env.db, email) })

		w := doRegister(env.router, name, email, password)
		if w.Code != http.StatusCreated {
			rt.Fatalf("registration failed: %d %s", w.Code, w.Body.String())
		}
		wl := doLoginWithIP(env.router, email, password, ip)
		if wl.Code != http.StatusOK {
			rt.Fatalf("login failed: %d %s", wl.Code, wl.Body.String())
		}

		jwtCookie := extractCookie(wl, "jwt")
		if jwtCookie == nil {
			rt.Fatal("no 'jwt' cookie set on login response")
		}
		if !jwtCookie.HttpOnly {
			rt.Fatal("jwt cookie must be HttpOnly")
		}
		if !jwtCookie.Secure {
			rt.Fatal("jwt cookie must have Secure=true")
		}
		if jwtCookie.SameSite != http.SameSiteStrictMode {
			rt.Fatalf("jwt cookie SameSite must be Strict, got %v", jwtCookie.SameSite)
		}
		if jwtCookie.MaxAge != 86400 {
			rt.Fatalf("jwt cookie MaxAge must be 86400, got %d", jwtCookie.MaxAge)
		}
		if jwtPattern.MatchString(wl.Body.String()) {
			rt.Fatalf("response body contains a JWT — must only be in HttpOnly cookie. Body: %s", wl.Body.String())
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Property 5: Wrong credentials always produce generic HTTP 401
// Validates: Requirements 2.6
// ─────────────────────────────────────────────────────────────────────────────

func TestProperty5_WrongCredentialsGenericHTTP401(t *testing.T) {
	env := setupTestEnv(t)
	rapid.Check(t, func(rt *rapid.T) {
		name := validNameGen().Draw(rt, "name")
		email := uniqueEmailGen().Draw(rt, "email")
		password := validPasswordGen().Draw(rt, "password")
		ip := uniqueIPGen().Draw(rt, "ip")
		t.Cleanup(func() { cleanupEmail(env.db, email) })

		w := doRegister(env.router, name, email, password)
		if w.Code != http.StatusCreated {
			rt.Fatalf("registration failed: %d %s", w.Code, w.Body.String())
		}

		// Wrong password.
		wl := doLoginWithIP(env.router, email, password+"_wrong", ip)
		if wl.Code != http.StatusUnauthorized {
			rt.Fatalf("wrong password: expected 401, got %d: %s", wl.Code, wl.Body.String())
		}
		var resp map[string]interface{}
		_ = json.Unmarshal(wl.Body.Bytes(), &resp)
		errMsg := strings.ToLower(fmt.Sprintf("%v", resp["error"]))
		if strings.Contains(errMsg, "password incorrect") || strings.Contains(errMsg, "email not found") {
			rt.Fatalf("error message reveals which field is wrong: %s", errMsg)
		}

		// Non-existent email.
		wl2 := doLoginWithIP(env.router, "nonexistent_"+email, password, ip)
		if wl2.Code != http.StatusUnauthorized {
			rt.Fatalf("non-existent email: expected 401, got %d: %s", wl2.Code, wl2.Body.String())
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Property 6: Requests to protected endpoints without valid JWT rejected with HTTP 401
// Validates: Requirements 3.1, 3.2
// ─────────────────────────────────────────────────────────────────────────────

func TestProperty6_ProtectedEndpointsRequireJWT(t *testing.T) {
	env := setupTestEnv(t)
	rapid.Check(t, func(rt *rapid.T) {
		// No cookie.
		req := httptest.NewRequest(http.MethodGet, "/api/protected/ping", nil)
		w := httptest.NewRecorder()
		env.router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			rt.Fatalf("no JWT: expected 401, got %d: %s", w.Code, w.Body.String())
		}

		// Invalid/tampered JWT cookie.
		req2 := httptest.NewRequest(http.MethodGet, "/api/protected/ping", nil)
		req2.AddCookie(&http.Cookie{Name: "jwt", Value: "invalid.jwt.token"})
		w2 := httptest.NewRecorder()
		env.router.ServeHTTP(w2, req2)
		if w2.Code != http.StatusUnauthorized {
			rt.Fatalf("invalid JWT: expected 401, got %d: %s", w2.Code, w2.Body.String())
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Property 7: CSRF token always unique and at least 64 chars
// Validates: Requirements 3.3, 3.4
// ─────────────────────────────────────────────────────────────────────────────

func TestProperty7_CSRFTokenUniqueAndMinLength64(t *testing.T) {
	env := setupTestEnv(t)
	rapid.Check(t, func(rt *rapid.T) {
		// 2–3 sessions per run: enough to verify uniqueness without being slow.
		n := rapid.IntRange(2, 3).Draw(rt, "n")
		name := validNameGen().Draw(rt, "name")
		email := uniqueEmailGen().Draw(rt, "email")
		password := validPasswordGen().Draw(rt, "password")
		ip := uniqueIPGen().Draw(rt, "ip")
		t.Cleanup(func() { cleanupEmail(env.db, email) })

		w := doRegister(env.router, name, email, password)
		if w.Code != http.StatusCreated {
			rt.Fatalf("registration failed: %d %s", w.Code, w.Body.String())
		}

		seen := make(map[string]struct{}, n)
		for i := 0; i < n; i++ {
			wl := doLoginWithIP(env.router, email, password, ip)
			if wl.Code != http.StatusOK {
				rt.Fatalf("login %d failed: %d %s", i, wl.Code, wl.Body.String())
			}
			c := extractCookie(wl, "csrf_token")
			if c == nil {
				rt.Fatalf("login %d: no csrf_token cookie", i)
			}
			if len(c.Value) < 64 {
				rt.Fatalf("login %d: CSRF token length %d < 64: %q", i, len(c.Value), c.Value)
			}
			if _, dup := seen[c.Value]; dup {
				rt.Fatalf("login %d: CSRF token not unique: %q", i, c.Value)
			}
			seen[c.Value] = struct{}{}
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Property 8: All mutating requests without valid CSRF token rejected with HTTP 403
// Validates: Requirements 2.6, 3.2, 3.3
// ─────────────────────────────────────────────────────────────────────────────

func TestProperty8_MutatingRequestsRequireCSRFToken(t *testing.T) {
	env := setupTestEnv(t)
	mutatingMethods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}

	rapid.Check(t, func(rt *rapid.T) {
		name := validNameGen().Draw(rt, "name")
		email := uniqueEmailGen().Draw(rt, "email")
		password := validPasswordGen().Draw(rt, "password")
		ip := uniqueIPGen().Draw(rt, "ip")
		t.Cleanup(func() { cleanupEmail(env.db, email) })

		w := doRegister(env.router, name, email, password)
		if w.Code != http.StatusCreated {
			rt.Fatalf("registration failed: %d %s", w.Code, w.Body.String())
		}
		wl := doLoginWithIP(env.router, email, password, ip)
		if wl.Code != http.StatusOK {
			rt.Fatalf("login failed: %d %s", wl.Code, wl.Body.String())
		}

		jwtCookie := extractCookie(wl, "jwt")
		csrfCookie := extractCookie(wl, "csrf_token")
		if jwtCookie == nil || csrfCookie == nil {
			rt.Fatal("missing jwt or csrf_token cookie after login")
		}

		method := rapid.SampledFrom(mutatingMethods).Draw(rt, "method")

		// Valid JWT, no CSRF header — must return 403.
		req := httptest.NewRequest(method, "/api/protected/ping", nil)
		req.AddCookie(&http.Cookie{Name: "jwt", Value: jwtCookie.Value})
		req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfCookie.Value})
		w1 := httptest.NewRecorder()
		env.router.ServeHTTP(w1, req)
		if w1.Code != http.StatusForbidden {
			rt.Fatalf("no CSRF header (%s): expected 403, got %d: %s", method, w1.Code, w1.Body.String())
		}

		// Valid JWT, wrong CSRF header — must return 403.
		req2 := httptest.NewRequest(method, "/api/protected/ping", nil)
		req2.AddCookie(&http.Cookie{Name: "jwt", Value: jwtCookie.Value})
		req2.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfCookie.Value})
		req2.Header.Set("X-CSRF-Token", "wrong-csrf-token-value")
		w2 := httptest.NewRecorder()
		env.router.ServeHTTP(w2, req2)
		if w2.Code != http.StatusForbidden {
			rt.Fatalf("wrong CSRF header (%s): expected 403, got %d: %s", method, w2.Code, w2.Body.String())
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Property 1: Registration stores bcrypt hash with cost >= 12
