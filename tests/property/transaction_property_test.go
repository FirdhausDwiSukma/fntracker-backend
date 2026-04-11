// Package property contains property-based tests for the transaction feature.
//
// Validates: Requirements 5.2, 5.8
//
// Run with reduced checks for faster execution:
//
//	go test ./tests/property/ -run "TestProperty13|TestProperty14" -rapid.checks=10
package property

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"pgregory.net/rapid"
)

// ─────────────────────────────────────────────────────────────────────────────
// Transaction test helpers
// ─────────────────────────────────────────────────────────────────────────────

// transactionData holds the fields of a created transaction as returned by the API.
// The models.Transaction struct has no explicit json tags, so Go uses the struct
// field names directly (e.g. "ID", "CategoryID", "Amount", "Type", "Date").
type transactionData struct {
	ID          uint    `json:"ID"`
	CategoryID  uint    `json:"CategoryID"`
	Amount      float64 `json:"Amount"`
	Type        string  `json:"Type"`
	Description string  `json:"Description"`
	Date        string  `json:"Date"`
}

// doCreateTransaction performs POST /api/transactions.
func doCreateTransaction(router *gin.Engine, jwtVal, csrfVal string, categoryID uint, amount float64, txType, date, description string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]interface{}{
		"category_id": categoryID,
		"amount":      amount,
		"type":        txType,
		"date":        date,
		"description": description,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/transactions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "jwt", Value: jwtVal})
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	req.Header.Set("X-CSRF-Token", csrfVal)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// doGetTransactions performs GET /api/transactions with optional query params.
func doGetTransactions(router *gin.Engine, jwtVal, csrfVal string, queryParams map[string]string) *httptest.ResponseRecorder {
	url := "/api/transactions"
	if len(queryParams) > 0 {
		url += "?"
		first := true
		for k, v := range queryParams {
			if !first {
				url += "&"
			}
			url += k + "=" + v
			first = false
		}
	}
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: jwtVal})
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	req.Header.Set("X-CSRF-Token", csrfVal)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// parseCreatedTransaction extracts the transaction from a create response body.
func parseCreatedTransaction(body []byte) *transactionData {
	var resp struct {
		Data transactionData `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil
	}
	return &resp.Data
}

// parseTransactionList extracts the list of transactions from a GET response body.
func parseTransactionList(body []byte) []transactionData {
	var resp struct {
		Data []transactionData `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil
	}
	return resp.Data
}

// cleanupTransactions removes all transactions for a user by email.
func cleanupTransactions(t *testing.T, env *testEnv, email string) {
	t.Helper()
	var userID uint
	row := env.db.Raw("SELECT id FROM users WHERE email = ?", email).Row()
	if err := row.Scan(&userID); err != nil || userID == 0 {
		return
	}
	env.db.Exec("DELETE FROM transactions WHERE user_id = ?", userID)
}

// validAmountGen generates positive float64 amounts (0.01 – 9999999.99).
func validAmountGen() *rapid.Generator[float64] {
	return rapid.Custom(func(t *rapid.T) float64 {
		cents := rapid.IntRange(1, 999999999).Draw(t, "cents")
		return float64(cents) / 100.0
	})
}

// validDateGen generates a random date string in YYYY-MM-DD format within a
// reasonable range (2020-01-01 to 2025-12-31).
func validDateGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		year := rapid.IntRange(2020, 2025).Draw(t, "year")
		month := rapid.IntRange(1, 12).Draw(t, "month")
		// Use a safe day range to avoid invalid dates.
		day := rapid.IntRange(1, 28).Draw(t, "day")
		return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
	})
}

// validDescriptionGen generates a short description (0–100 printable ASCII chars).
func validDescriptionGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		length := rapid.IntRange(0, 50).Draw(t, "desc_len")
		if length == 0 {
			return ""
		}
		return rapid.StringOfN(
			rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 ")),
			length, length, -1,
		).Draw(t, "desc")
	})
}

// setupTransactionUser registers a user, logs in, and creates one category of
// each type. Returns jwt, csrf, incomeCategID, expenseCategID.
func setupTransactionUser(t *testing.T, rt *rapid.T, env *testEnv, email, name, password string) (jwtVal, csrfVal string, incomeCatID, expenseCatID uint, ok bool) {
	t.Helper()

	jwtVal, csrfVal, ok = registerAndLoginWithDB(t, env.router, env.db, name, email, password)
	if !ok {
		return "", "", 0, 0, false
	}

	// Create income category.
	wIncome := doCreateCategory(env.router, jwtVal, csrfVal, "Income Cat", "income")
	if wIncome.Code != http.StatusCreated {
		return "", "", 0, 0, false
	}
	incomeCatID = parseCategoryID(wIncome.Body.Bytes())

	// Create expense category.
	wExpense := doCreateCategory(env.router, jwtVal, csrfVal, "Expense Cat", "expense")
	if wExpense.Code != http.StatusCreated {
		return "", "", 0, 0, false
	}
	expenseCatID = parseCategoryID(wExpense.Body.Bytes())

	return jwtVal, csrfVal, incomeCatID, expenseCatID, true
}

// ─────────────────────────────────────────────────────────────────────────────
// Property 13: Pembuatan transaksi adalah operasi round-trip yang konsisten
// Validates: Requirements 5.2
// ─────────────────────────────────────────────────────────────────────────────

// TestProperty13_TransactionCreateRoundTrip verifies that for any valid transaction
// data, after a successful create, a GET returns the transaction with all fields
// matching the input.
//
// **Validates: Requirements 5.2**
func TestProperty13_TransactionCreateRoundTrip(t *testing.T) {
	env := setupCategoryEnv(t)

	rapid.Check(t, func(rt *rapid.T) {
		email := fmt.Sprintf("prop13%s@test.com", uniqueSuffix(rt, "suffix"))
		name := validNameGen().Draw(rt, "name")
		password := validPasswordGen().Draw(rt, "password")

		t.Cleanup(func() {
			cleanupTransactions(t, env, email)
			cleanupUser(env.db, email)
		})

		jwtVal, csrfVal, incomeCatID, expenseCatID, ok := setupTransactionUser(t, rt, env, email, name, password)
		if !ok {
			rt.Fatalf("failed to set up user and categories for %s", email)
		}

		// Draw random valid transaction fields.
		txType := validCategoryTypeGen().Draw(rt, "txType")
		var catID uint
		if txType == "income" {
			catID = incomeCatID
		} else {
			catID = expenseCatID
		}
		amount := validAmountGen().Draw(rt, "amount")
		date := validDateGen().Draw(rt, "date")
		description := validDescriptionGen().Draw(rt, "description")

		// Create the transaction.
		wCreate := doCreateTransaction(env.router, jwtVal, csrfVal, catID, amount, txType, date, description)
		if wCreate.Code != http.StatusCreated {
			rt.Fatalf("create transaction failed: %d %s", wCreate.Code, wCreate.Body.String())
		}

		created := parseCreatedTransaction(wCreate.Body.Bytes())
		if created == nil || created.ID == 0 {
			rt.Fatalf("could not parse created transaction from: %s", wCreate.Body.String())
		}

		// GET all transactions and find the one we just created.
		wGet := doGetTransactions(env.router, jwtVal, csrfVal, nil)
		if wGet.Code != http.StatusOK {
			rt.Fatalf("GET transactions failed: %d %s", wGet.Code, wGet.Body.String())
		}

		txList := parseTransactionList(wGet.Body.Bytes())
		var found *transactionData
		for i := range txList {
			if txList[i].ID == created.ID {
				found = &txList[i]
				break
			}
		}
		if found == nil {
			rt.Fatalf("created transaction ID %d not found in GET response", created.ID)
		}

		// Verify round-trip field consistency.
		if found.CategoryID != catID {
			rt.Fatalf("category_id mismatch: sent %d, got %d", catID, found.CategoryID)
		}
		if found.Type != txType {
			rt.Fatalf("type mismatch: sent %q, got %q", txType, found.Type)
		}
		// Amount: compare with tolerance for decimal representation.
		const epsilon = 0.005
		if diff := found.Amount - amount; diff > epsilon || diff < -epsilon {
			rt.Fatalf("amount mismatch: sent %.2f, got %.2f", amount, found.Amount)
		}
		// Date: the API returns a full RFC3339 timestamp; check the date prefix.
		if len(found.Date) < 10 || found.Date[:10] != date {
			rt.Fatalf("date mismatch: sent %q, got %q", date, found.Date)
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Property 14: Filter transaksi selalu menghasilkan subset yang memenuhi kriteria
// Validates: Requirements 5.8
// ─────────────────────────────────────────────────────────────────────────────

// TestProperty14_TransactionFilterSubsetCorrectness verifies that for any filter
// criteria applied to GET /api/transactions, every returned transaction satisfies
// ALL applied filter criteria.
//
// **Validates: Requirements 5.8**
func TestProperty14_TransactionFilterSubsetCorrectness(t *testing.T) {
	env := setupCategoryEnv(t)

	rapid.Check(t, func(rt *rapid.T) {
		email := fmt.Sprintf("prop14%s@test.com", uniqueSuffix(rt, "suffix"))
		name := validNameGen().Draw(rt, "name")
		password := validPasswordGen().Draw(rt, "password")

		t.Cleanup(func() {
			cleanupTransactions(t, env, email)
			cleanupUser(env.db, email)
		})

		jwtVal, csrfVal, incomeCatID, expenseCatID, ok := setupTransactionUser(t, rt, env, email, name, password)
		if !ok {
			rt.Fatalf("failed to set up user and categories for %s", email)
		}

		// Create a small set of transactions with varied types and dates.
		type txSeed struct {
			catID  uint
			txType string
			date   string
		}
		seeds := []txSeed{
			{incomeCatID, "income", "2023-03-15"},
			{expenseCatID, "expense", "2023-03-20"},
			{incomeCatID, "income", "2023-06-10"},
			{expenseCatID, "expense", "2023-06-25"},
			{incomeCatID, "income", "2024-01-05"},
			{expenseCatID, "expense", "2024-01-18"},
		}

		for _, s := range seeds {
			amount := validAmountGen().Draw(rt, fmt.Sprintf("amount_%s_%s", s.txType, s.date))
			w := doCreateTransaction(env.router, jwtVal, csrfVal, s.catID, amount, s.txType, s.date, "test")
			if w.Code != http.StatusCreated {
				rt.Fatalf("seed transaction create failed (%s %s): %d %s", s.txType, s.date, w.Code, w.Body.String())
			}
		}

		// ── Sub-test A: filter by type ────────────────────────────────────────
		filterType := validCategoryTypeGen().Draw(rt, "filterType")
		wType := doGetTransactions(env.router, jwtVal, csrfVal, map[string]string{"type": filterType, "limit": "100"})
		if wType.Code != http.StatusOK {
			rt.Fatalf("GET with type filter failed: %d %s", wType.Code, wType.Body.String())
		}
		for _, tx := range parseTransactionList(wType.Body.Bytes()) {
			if tx.Type != filterType {
				rt.Fatalf("type filter %q: got transaction with type %q (ID %d)", filterType, tx.Type, tx.ID)
			}
		}

		// ── Sub-test B: filter by category_id ────────────────────────────────
		filterCatID := incomeCatID
		if rapid.Bool().Draw(rt, "use_expense_cat") {
			filterCatID = expenseCatID
		}
		wCat := doGetTransactions(env.router, jwtVal, csrfVal, map[string]string{
			"category_id": fmt.Sprintf("%d", filterCatID),
			"limit":       "100",
		})
		if wCat.Code != http.StatusOK {
			rt.Fatalf("GET with category_id filter failed: %d %s", wCat.Code, wCat.Body.String())
		}
		for _, tx := range parseTransactionList(wCat.Body.Bytes()) {
			if tx.CategoryID != filterCatID {
				rt.Fatalf("category_id filter %d: got transaction with category_id %d (ID %d)", filterCatID, tx.CategoryID, tx.ID)
			}
		}

		// ── Sub-test C: filter by start_date / end_date ───────────────────────
		startDate := "2023-06-01"
		endDate := "2023-06-30"
		wDate := doGetTransactions(env.router, jwtVal, csrfVal, map[string]string{
			"start_date": startDate,
			"end_date":   endDate,
			"limit":      "100",
		})
		if wDate.Code != http.StatusOK {
			rt.Fatalf("GET with date range filter failed: %d %s", wDate.Code, wDate.Body.String())
		}
		start, _ := time.Parse("2006-01-02", startDate)
		end, _ := time.Parse("2006-01-02", endDate)
		for _, tx := range parseTransactionList(wDate.Body.Bytes()) {
			txDate, err := time.Parse(time.RFC3339, tx.Date)
			if err != nil {
				// Try plain date format as fallback.
				txDate, err = time.Parse("2006-01-02", tx.Date[:10])
				if err != nil {
					rt.Fatalf("cannot parse transaction date %q: %v", tx.Date, err)
				}
			}
			txDateOnly := txDate.Truncate(24 * time.Hour)
			if txDateOnly.Before(start) || txDateOnly.After(end) {
				rt.Fatalf("date range filter [%s, %s]: got transaction with date %s (ID %d)",
					startDate, endDate, tx.Date[:10], tx.ID)
			}
		}

		// ── Sub-test D: filter by month and year ──────────────────────────────
		filterMonth := 3
		filterYear := 2023
		wMonthYear := doGetTransactions(env.router, jwtVal, csrfVal, map[string]string{
			"month": fmt.Sprintf("%d", filterMonth),
			"year":  fmt.Sprintf("%d", filterYear),
			"limit": "100",
		})
		if wMonthYear.Code != http.StatusOK {
			rt.Fatalf("GET with month/year filter failed: %d %s", wMonthYear.Code, wMonthYear.Body.String())
		}
		for _, tx := range parseTransactionList(wMonthYear.Body.Bytes()) {
			txDate, err := time.Parse(time.RFC3339, tx.Date)
			if err != nil {
				txDate, err = time.Parse("2006-01-02", tx.Date[:10])
				if err != nil {
					rt.Fatalf("cannot parse transaction date %q: %v", tx.Date, err)
				}
			}
			if int(txDate.Month()) != filterMonth || txDate.Year() != filterYear {
				rt.Fatalf("month/year filter %d/%d: got transaction with date %s (ID %d)",
					filterMonth, filterYear, tx.Date[:10], tx.ID)
			}
		}
	})
}
