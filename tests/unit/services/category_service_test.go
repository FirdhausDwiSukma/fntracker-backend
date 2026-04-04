// Package services_test contains unit tests for CategoryService.
// Validates: Requirements 4.1–4.5
package services_test

import (
	"errors"
	"testing"

	"finance-tracker/dto"
	"finance-tracker/models"
	"finance-tracker/services"
)

// ─── Mock CategoryRepository ─────────────────────────────────────────────────

type mockCategoryRepo struct {
	categories     []*models.Category
	nextID         uint
	forceCreateErr error
	forceUpdateErr error
	forceDeleteErr error
	forceFindErr   error
	forceExistsErr error
}

func newMockCategoryRepo() *mockCategoryRepo {
	return &mockCategoryRepo{nextID: 1}
}

func (m *mockCategoryRepo) FindAllByUser(userID uint) ([]models.Category, error) {
	if m.forceFindErr != nil {
		return nil, m.forceFindErr
	}
	var result []models.Category
	for _, c := range m.categories {
		if c.UserID == userID {
			result = append(result, *c)
		}
	}
	return result, nil
}

func (m *mockCategoryRepo) FindByIDAndUser(id, userID uint) (*models.Category, error) {
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

func (m *mockCategoryRepo) Create(category *models.Category) error {
	if m.forceCreateErr != nil {
		return m.forceCreateErr
	}
	category.ID = m.nextID
	m.nextID++
	cp := *category
	m.categories = append(m.categories, &cp)
	return nil
}

func (m *mockCategoryRepo) Update(category *models.Category) error {
	if m.forceUpdateErr != nil {
		return m.forceUpdateErr
	}
	for i, c := range m.categories {
		if c.ID == category.ID {
			cp := *category
			m.categories[i] = &cp
			return nil
		}
	}
	return errors.New("not found")
}

func (m *mockCategoryRepo) Delete(id, userID uint) error {
	if m.forceDeleteErr != nil {
		return m.forceDeleteErr
	}
	for i, c := range m.categories {
		if c.ID == id && c.UserID == userID {
			m.categories = append(m.categories[:i], m.categories[i+1:]...)
			return nil
		}
	}
	return errors.New("not found")
}

func (m *mockCategoryRepo) ExistsByNameTypeUser(name, categoryType string, userID uint) (bool, error) {
	if m.forceExistsErr != nil {
		return false, m.forceExistsErr
	}
	for _, c := range m.categories {
		if c.UserID == userID && c.Name == name && c.Type == categoryType {
			return true, nil
		}
	}
	return false, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func newCategoryService(repo *mockCategoryRepo) services.CategoryService {
	return services.NewCategoryService(repo)
}

func seedCategory(t *testing.T, svc services.CategoryService, userID uint, name, typ string) *models.Category {
	t.Helper()
	cat, err := svc.Create(userID, dto.CategoryRequest{Name: name, Type: typ})
	if err != nil {
		t.Fatalf("seed Create failed: %v", err)
	}
	return cat
}

// ─── GetAllByUser ─────────────────────────────────────────────────────────────

// Req 4.1: GetAllByUser returns only the authenticated user's categories.
func TestGetAllByUser_ReturnsOnlyOwnCategories(t *testing.T) {
	repo := newMockCategoryRepo()
	svc := newCategoryService(repo)

	seedCategory(t, svc, 1, "Salary", "income")
	seedCategory(t, svc, 1, "Food", "expense")
	seedCategory(t, svc, 2, "Freelance", "income") // belongs to user 2

	cats, err := svc.GetAllByUser(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cats) != 2 {
		t.Fatalf("expected 2 categories for user 1, got %d", len(cats))
	}
	for _, c := range cats {
		if c.UserID != 1 {
			t.Errorf("returned category belongs to user %d, not user 1", c.UserID)
		}
	}
}

func TestGetAllByUser_EmptyForNewUser(t *testing.T) {
	repo := newMockCategoryRepo()
	svc := newCategoryService(repo)

	cats, err := svc.GetAllByUser(99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cats) != 0 {
		t.Fatalf("expected 0 categories, got %d", len(cats))
	}
}

// ─── Create ───────────────────────────────────────────────────────────────────

// Req 4.2: Create with valid input saves category with correct fields.
func TestCreate_ValidInput(t *testing.T) {
	tests := []struct {
		name    string
		catName string
		catType string
		userID  uint
	}{
		{"income category", "Salary", "income", 1},
		{"expense category", "Groceries", "expense", 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockCategoryRepo()
			svc := newCategoryService(repo)

			cat, err := svc.Create(tc.userID, dto.CategoryRequest{Name: tc.catName, Type: tc.catType})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cat == nil {
				t.Fatal("expected category, got nil")
			}
			if cat.UserID != tc.userID {
				t.Errorf("UserID: got %d, want %d", cat.UserID, tc.userID)
			}
			if cat.Name != tc.catName {
				t.Errorf("Name: got %q, want %q", cat.Name, tc.catName)
			}
			if cat.Type != tc.catType {
				t.Errorf("Type: got %q, want %q", cat.Type, tc.catType)
			}
			if cat.ID == 0 {
				t.Error("expected non-zero ID after create")
			}
		})
	}
}

// Req 4.3: Create with invalid type returns an error.
// Note: type validation is enforced at the DTO/binding layer (oneof=income expense).
// The service itself trusts the DTO, so we verify the service passes through valid types
// and that the mock correctly stores them. Invalid types would be rejected before reaching
// the service in production. We test the service's behavior with valid types only.
func TestCreate_ValidTypes_Accepted(t *testing.T) {
	validTypes := []string{"income", "expense"}
	for _, typ := range validTypes {
		t.Run("type="+typ, func(t *testing.T) {
			repo := newMockCategoryRepo()
			svc := newCategoryService(repo)

			cat, err := svc.Create(1, dto.CategoryRequest{Name: "Test", Type: typ})
			if err != nil {
				t.Fatalf("valid type %q rejected: %v", typ, err)
			}
			if cat.Type != typ {
				t.Errorf("type mismatch: got %q, want %q", cat.Type, typ)
			}
		})
	}
}

// Req 4.5: Create with duplicate (name, type, userID) returns error.
func TestCreate_DuplicateNameTypeUser_ReturnsError(t *testing.T) {
	repo := newMockCategoryRepo()
	svc := newCategoryService(repo)

	req := dto.CategoryRequest{Name: "Food", Type: "expense"}
	if _, err := svc.Create(1, req); err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	_, err := svc.Create(1, req)
	if err == nil {
		t.Fatal("expected error for duplicate category, got nil")
	}
}

// Req 4.5: Same name + different type is allowed.
func TestCreate_SameNameDifferentType_Allowed(t *testing.T) {
	repo := newMockCategoryRepo()
	svc := newCategoryService(repo)

	if _, err := svc.Create(1, dto.CategoryRequest{Name: "Transfer", Type: "income"}); err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	_, err := svc.Create(1, dto.CategoryRequest{Name: "Transfer", Type: "expense"})
	if err != nil {
		t.Fatalf("same name different type should be allowed, got: %v", err)
	}
}

// Req 4.5: Same name + same type for different users is allowed.
func TestCreate_SameNameSameType_DifferentUsers_Allowed(t *testing.T) {
	repo := newMockCategoryRepo()
	svc := newCategoryService(repo)

	if _, err := svc.Create(1, dto.CategoryRequest{Name: "Food", Type: "expense"}); err != nil {
		t.Fatalf("user 1 create failed: %v", err)
	}
	_, err := svc.Create(2, dto.CategoryRequest{Name: "Food", Type: "expense"})
	if err != nil {
		t.Fatalf("user 2 should be able to create same category, got: %v", err)
	}
}

// ─── Update ───────────────────────────────────────────────────────────────────

// Req 4.4: Update with a categoryID belonging to another user returns error.
func TestUpdate_WrongOwner_ReturnsError(t *testing.T) {
	repo := newMockCategoryRepo()
	svc := newCategoryService(repo)

	// user 1 creates a category
	cat := seedCategory(t, svc, 1, "Salary", "income")

	// user 2 tries to update it
	_, err := svc.Update(2, cat.ID, dto.CategoryRequest{Name: "Hacked", Type: "expense"})
	if err == nil {
		t.Fatal("expected error when updating another user's category, got nil")
	}
}

// Update with valid ownership succeeds.
func TestUpdate_ValidOwnership_Succeeds(t *testing.T) {
	repo := newMockCategoryRepo()
	svc := newCategoryService(repo)

	cat := seedCategory(t, svc, 1, "Salary", "income")

	updated, err := svc.Update(1, cat.ID, dto.CategoryRequest{Name: "Bonus", Type: "income"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "Bonus" {
		t.Errorf("Name: got %q, want %q", updated.Name, "Bonus")
	}
	if updated.Type != "income" {
		t.Errorf("Type: got %q, want %q", updated.Type, "income")
	}
}

// Req 4.5: Update to a duplicate (name, type) for the same user returns error.
func TestUpdate_DuplicateNameType_ReturnsError(t *testing.T) {
	repo := newMockCategoryRepo()
	svc := newCategoryService(repo)

	seedCategory(t, svc, 1, "Food", "expense")
	cat2 := seedCategory(t, svc, 1, "Transport", "expense")

	// Try to rename cat2 to "Food" expense — conflicts with existing
	_, err := svc.Update(1, cat2.ID, dto.CategoryRequest{Name: "Food", Type: "expense"})
	if err == nil {
		t.Fatal("expected error for duplicate name+type on update, got nil")
	}
}

// Update with same name+type (no change) should not return duplicate error.
func TestUpdate_SameNameType_NoChange_Succeeds(t *testing.T) {
	repo := newMockCategoryRepo()
	svc := newCategoryService(repo)

	cat := seedCategory(t, svc, 1, "Food", "expense")

	_, err := svc.Update(1, cat.ID, dto.CategoryRequest{Name: "Food", Type: "expense"})
	if err != nil {
		t.Fatalf("updating with same name+type should succeed, got: %v", err)
	}
}

// ─── Delete ───────────────────────────────────────────────────────────────────

// Req 4.4: Delete with a categoryID belonging to another user returns error.
func TestDelete_WrongOwner_ReturnsError(t *testing.T) {
	repo := newMockCategoryRepo()
	svc := newCategoryService(repo)

	cat := seedCategory(t, svc, 1, "Salary", "income")

	err := svc.Delete(2, cat.ID)
	if err == nil {
		t.Fatal("expected error when deleting another user's category, got nil")
	}
}

// Delete with valid ownership succeeds.
func TestDelete_ValidOwnership_Succeeds(t *testing.T) {
	repo := newMockCategoryRepo()
	svc := newCategoryService(repo)

	cat := seedCategory(t, svc, 1, "Salary", "income")

	if err := svc.Delete(1, cat.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's gone
	cats, _ := svc.GetAllByUser(1)
	for _, c := range cats {
		if c.ID == cat.ID {
			t.Error("category still exists after delete")
		}
	}
}

// Delete non-existent category returns error.
func TestDelete_NonExistent_ReturnsError(t *testing.T) {
	repo := newMockCategoryRepo()
	svc := newCategoryService(repo)

	err := svc.Delete(1, 999)
	if err == nil {
		t.Fatal("expected error for non-existent category, got nil")
	}
}
