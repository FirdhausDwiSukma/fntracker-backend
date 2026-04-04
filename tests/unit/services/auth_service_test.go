// Package services_test contains unit tests for AuthService.
// Validates: Requirements 1.1–1.6, 2.1–2.5
package services_test

import (
	"errors"
	"testing"

	"finance-tracker/dto"
	"finance-tracker/models"
	"finance-tracker/services"

	"golang.org/x/crypto/bcrypt"
)

// mockUserRepo is an in-memory UserRepository for unit testing.
type mockUserRepo struct {
	users  map[string]*models.User
	nextID uint
	// forceCreateErr makes Create return an error.
	forceCreateErr error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*models.User), nextID: 1}
}

func (m *mockUserRepo) Create(user *models.User) error {
	if m.forceCreateErr != nil {
		return m.forceCreateErr
	}
	user.ID = m.nextID
	m.nextID++
	// Store a copy.
	cp := *user
	m.users[user.Email] = &cp
	return nil
}

func (m *mockUserRepo) FindByEmail(email string) (*models.User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func (m *mockUserRepo) FindByID(id uint) (*models.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			cp := *u
			return &cp, nil
		}
	}
	return nil, nil
}

const secret = "unit-test-jwt-secret-32-chars-ok"

// ─── Register ────────────────────────────────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	repo := newMockUserRepo()
	svc := services.NewAuthService(repo, secret)

	user, err := svc.Register(dto.RegisterRequest{
		Name:     "Alice",
		Email:    "alice@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if user.Email != "alice@example.com" {
		t.Errorf("email mismatch: got %q", user.Email)
	}
	// Password must be cleared in the returned struct.
	if user.Password != "" {
		t.Errorf("password should be cleared in response, got %q", user.Password)
	}
}

func TestRegister_PasswordStoredAsHash(t *testing.T) {
	repo := newMockUserRepo()
	svc := services.NewAuthService(repo, secret)

	plaintext := "securepassword"
	_, err := svc.Register(dto.RegisterRequest{
		Name:     "Bob",
		Email:    "bob@example.com",
		Password: plaintext,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored := repo.users["bob@example.com"]
	if stored.Password == plaintext {
		t.Fatal("password stored as plaintext — must be hashed")
	}
	cost, err := bcrypt.Cost([]byte(stored.Password))
	if err != nil {
		t.Fatalf("stored value is not a valid bcrypt hash: %v", err)
	}
	if cost < 12 {
		t.Fatalf("bcrypt cost %d < required minimum 12", cost)
	}
}

func TestRegister_NameSanitized(t *testing.T) {
	repo := newMockUserRepo()
	svc := services.NewAuthService(repo, secret)

	user, err := svc.Register(dto.RegisterRequest{
		Name:     "<script>alert(1)</script>",
		Email:    "xss@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Name == "<script>alert(1)</script>" {
		t.Error("name was not sanitized")
	}
}

func TestRegister_DuplicateEmailReturnsError(t *testing.T) {
	repo := newMockUserRepo()
	svc := services.NewAuthService(repo, secret)

	req := dto.RegisterRequest{Name: "Alice", Email: "dup@example.com", Password: "password123"}
	if _, err := svc.Register(req); err != nil {
		t.Fatalf("first registration failed: %v", err)
	}
	_, err := svc.Register(req)
	if err == nil {
		t.Fatal("expected error for duplicate email, got nil")
	}
}

func TestRegister_RepoCreateErrorPropagated(t *testing.T) {
	repo := newMockUserRepo()
	repo.forceCreateErr = errors.New("db error")
	svc := services.NewAuthService(repo, secret)

	_, err := svc.Register(dto.RegisterRequest{
		Name:     "Carol",
		Email:    "carol@example.com",
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
}

// ─── Login ───────────────────────────────────────────────────────────────────

func registerUser(t *testing.T, svc services.AuthService, email, password string) {
	t.Helper()
	_, err := svc.Register(dto.RegisterRequest{Name: "Test", Email: email, Password: password})
	if err != nil {
		t.Fatalf("setup register failed: %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	repo := newMockUserRepo()
	svc := services.NewAuthService(repo, secret)
	registerUser(t, svc, "login@example.com", "password123")

	resp, err := svc.Login(dto.LoginRequest{Email: "login@example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Token == "" {
		t.Error("expected non-empty JWT token")
	}
	if resp.CsrfToken == "" {
		t.Error("expected non-empty CSRF token")
	}
	if resp.User == nil {
		t.Error("expected user in response")
	}
	if resp.User.Password != "" {
		t.Error("password should be cleared in login response")
	}
}

func TestLogin_WrongPasswordReturnsError(t *testing.T) {
	repo := newMockUserRepo()
	svc := services.NewAuthService(repo, secret)
	registerUser(t, svc, "wp@example.com", "correctpassword")

	_, err := svc.Login(dto.LoginRequest{Email: "wp@example.com", Password: "wrongpassword"})
	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
	// Error message must be generic — must not reveal which field is wrong.
	if err.Error() == "password incorrect" || err.Error() == "email not found" {
		t.Errorf("error message reveals field: %q", err.Error())
	}
}

func TestLogin_NonExistentEmailReturnsError(t *testing.T) {
	repo := newMockUserRepo()
	svc := services.NewAuthService(repo, secret)

	_, err := svc.Login(dto.LoginRequest{Email: "ghost@example.com", Password: "password123"})
	if err == nil {
		t.Fatal("expected error for non-existent email, got nil")
	}
}

func TestLogin_TokenIsValidJWT(t *testing.T) {
	repo := newMockUserRepo()
	svc := services.NewAuthService(repo, secret)
	registerUser(t, svc, "jwt@example.com", "password123")

	resp, _ := svc.Login(dto.LoginRequest{Email: "jwt@example.com", Password: "password123"})

	// Token must have three dot-separated segments.
	parts := splitDots(resp.Token)
	if len(parts) != 3 {
		t.Fatalf("JWT must have 3 parts, got %d: %q", len(parts), resp.Token)
	}
}

func TestLogin_CSRFTokenLength(t *testing.T) {
	repo := newMockUserRepo()
	svc := services.NewAuthService(repo, secret)
	registerUser(t, svc, "csrf@example.com", "password123")

	resp, _ := svc.Login(dto.LoginRequest{Email: "csrf@example.com", Password: "password123"})
	if len(resp.CsrfToken) < 64 {
		t.Fatalf("CSRF token length %d < 64", len(resp.CsrfToken))
	}
}

// splitDots splits a string by "." without importing strings in test output.
func splitDots(s string) []string {
	var parts []string
	start := 0
	for i, c := range s {
		if c == '.' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}
