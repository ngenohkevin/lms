package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/services"
)

func TestSearchTransactions_ByQuery(t *testing.T) {
	// Set up test database
	db := setupTestDB(t)
	defer db.Close()

	querier := queries.New(db)
	transactionService := services.NewTransactionService(querier)
	ctx := context.Background()

	// Create test data with unique identifiers
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)
	student := createTestStudent(t, querier, "SearchQuery", "Student", "STU_SQ_"+suffix)
	librarian := createTestLibrarian(t, querier, "search_librarian_"+suffix, "search.lib."+suffix+"@example.com")
	book := createTestBook(t, querier, "Searchable Fantasy Book", "Fantasy Author", "BK_SQ_"+suffix, 1)

	// Create a transaction
	_, err := transactionService.BorrowBook(ctx, student.ID, book.ID, librarian.ID, "Search test")
	require.NoError(t, err)

	// Test: Search by book title
	t.Run("SearchByBookTitle", func(t *testing.T) {
		result, err := transactionService.SearchTransactions(ctx, services.TransactionSearchParams{
			Query: "Fantasy",
			Page:  1,
			Limit: 20,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result.Transactions), 1)

		found := false
		for _, tx := range result.Transactions {
			if tx.BookTitle == "Searchable Fantasy Book" {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected to find the test book in search results")
	})

	// Test: Search by author
	t.Run("SearchByAuthor", func(t *testing.T) {
		result, err := transactionService.SearchTransactions(ctx, services.TransactionSearchParams{
			Query: "Fantasy Author",
			Page:  1,
			Limit: 20,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result.Transactions), 1)
	})

	// Test: Search by student name
	t.Run("SearchByStudentName", func(t *testing.T) {
		result, err := transactionService.SearchTransactions(ctx, services.TransactionSearchParams{
			Query: "SearchQuery",
			Page:  1,
			Limit: 20,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result.Transactions), 1)
	})

	// Test: No results
	t.Run("NoResults", func(t *testing.T) {
		result, err := transactionService.SearchTransactions(ctx, services.TransactionSearchParams{
			Query: "NonExistentXYZ123",
			Page:  1,
			Limit: 20,
		})
		require.NoError(t, err)
		assert.Len(t, result.Transactions, 0)
	})
}

func TestSearchTransactions_ByStatus(t *testing.T) {
	// Set up test database
	db := setupTestDB(t)
	defer db.Close()

	querier := queries.New(db)
	transactionService := services.NewTransactionService(querier)
	ctx := context.Background()

	// Create test data
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)
	student := createTestStudent(t, querier, "StatusSearch", "Student", "STU_SS_"+suffix)
	librarian := createTestLibrarian(t, querier, "status_librarian_"+suffix, "status.lib."+suffix+"@example.com")
	activeBook := createTestBook(t, querier, "Active Book", "Author", "BK_ACT_"+suffix, 1)
	returnedBook := createTestBook(t, querier, "Returned Book", "Author", "BK_RET_"+suffix, 1)

	// Create an active transaction (not returned)
	_, err := transactionService.BorrowBook(ctx, student.ID, activeBook.ID, librarian.ID, "Active test")
	require.NoError(t, err)

	// Create and return a transaction
	tx, err := transactionService.BorrowBook(ctx, student.ID, returnedBook.ID, librarian.ID, "Return test")
	require.NoError(t, err)

	_, err = transactionService.ReturnBook(ctx, tx.ID)
	require.NoError(t, err)

	// Test: Filter by active status
	t.Run("FilterByActiveStatus", func(t *testing.T) {
		result, err := transactionService.SearchTransactions(ctx, services.TransactionSearchParams{
			Status: "active",
			Page:   1,
			Limit:  100,
		})
		require.NoError(t, err)

		// All results should have active status
		for _, tx := range result.Transactions {
			assert.Equal(t, "active", tx.Status)
		}
	})

	// Test: Filter by returned status
	t.Run("FilterByReturnedStatus", func(t *testing.T) {
		result, err := transactionService.SearchTransactions(ctx, services.TransactionSearchParams{
			Status: "returned",
			Page:   1,
			Limit:  100,
		})
		require.NoError(t, err)

		// All results should have returned status
		for _, tx := range result.Transactions {
			assert.Equal(t, "returned", tx.Status)
		}
	})
}

func TestSearchTransactions_ByDateRange(t *testing.T) {
	// Set up test database
	db := setupTestDB(t)
	defer db.Close()

	querier := queries.New(db)
	transactionService := services.NewTransactionService(querier)
	ctx := context.Background()

	// Create test data
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)
	student := createTestStudent(t, querier, "DateSearch", "Student", "STU_DS_"+suffix)
	librarian := createTestLibrarian(t, querier, "date_librarian_"+suffix, "date.lib."+suffix+"@example.com")
	book := createTestBook(t, querier, "Date Test Book", "Author", "BK_DT_"+suffix, 1)

	// Create a transaction
	_, err := transactionService.BorrowBook(ctx, student.ID, book.ID, librarian.ID, "Date range test")
	require.NoError(t, err)

	// Test: Filter by date range (today)
	t.Run("FilterByTodaysDate", func(t *testing.T) {
		now := time.Now()
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		endOfDay := startOfDay.Add(24 * time.Hour)

		result, err := transactionService.SearchTransactions(ctx, services.TransactionSearchParams{
			FromDate: &startOfDay,
			ToDate:   &endOfDay,
			Page:     1,
			Limit:    100,
		})
		require.NoError(t, err)

		// Should have results from today
		assert.GreaterOrEqual(t, len(result.Transactions), 1)
	})

	// Test: Filter by past date range (no results expected from yesterday)
	t.Run("FilterByPastDate", func(t *testing.T) {
		yesterday := time.Now().Add(-48 * time.Hour)
		dayBefore := yesterday.Add(-24 * time.Hour)

		result, err := transactionService.SearchTransactions(ctx, services.TransactionSearchParams{
			FromDate: &dayBefore,
			ToDate:   &yesterday,
			Page:     1,
			Limit:    100,
		})
		require.NoError(t, err)
		// We don't assert count as there might be old test data
		_ = result
	})
}

func TestSearchTransactions_ByStudentID(t *testing.T) {
	// Set up test database
	db := setupTestDB(t)
	defer db.Close()

	querier := queries.New(db)
	transactionService := services.NewTransactionService(querier)
	ctx := context.Background()

	// Create test data
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)
	student1 := createTestStudent(t, querier, "Student", "One", "STU_S1_"+suffix)
	student2 := createTestStudent(t, querier, "Student", "Two", "STU_S2_"+suffix)
	librarian := createTestLibrarian(t, querier, "student_librarian_"+suffix, "student.lib."+suffix+"@example.com")
	book1 := createTestBook(t, querier, "Book One", "Author", "BK_B1_"+suffix, 1)
	book2 := createTestBook(t, querier, "Book Two", "Author", "BK_B2_"+suffix, 1)

	// Create transactions for different students
	_, err := transactionService.BorrowBook(ctx, student1.ID, book1.ID, librarian.ID, "Student 1")
	require.NoError(t, err)

	_, err = transactionService.BorrowBook(ctx, student2.ID, book2.ID, librarian.ID, "Student 2")
	require.NoError(t, err)

	// Test: Filter by student 1
	t.Run("FilterByStudent1", func(t *testing.T) {
		result, err := transactionService.SearchTransactions(ctx, services.TransactionSearchParams{
			StudentID: &student1.ID,
			Page:      1,
			Limit:     20,
		})
		require.NoError(t, err)

		// All results should be for student 1
		for _, tx := range result.Transactions {
			assert.Equal(t, student1.ID, tx.StudentID)
		}
	})

	// Test: Filter by student 2
	t.Run("FilterByStudent2", func(t *testing.T) {
		result, err := transactionService.SearchTransactions(ctx, services.TransactionSearchParams{
			StudentID: &student2.ID,
			Page:      1,
			Limit:     20,
		})
		require.NoError(t, err)

		// All results should be for student 2
		for _, tx := range result.Transactions {
			assert.Equal(t, student2.ID, tx.StudentID)
		}
	})
}

func TestSearchTransactions_ByBookID(t *testing.T) {
	// Set up test database
	db := setupTestDB(t)
	defer db.Close()

	querier := queries.New(db)
	transactionService := services.NewTransactionService(querier)
	ctx := context.Background()

	// Create test data
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)
	student := createTestStudent(t, querier, "Book", "Search", "STU_BS_"+suffix)
	librarian := createTestLibrarian(t, querier, "book_librarian_"+suffix, "book.lib."+suffix+"@example.com")
	targetBook := createTestBook(t, querier, "Target Book", "Author", "BK_TG_"+suffix, 1)
	otherBook := createTestBook(t, querier, "Other Book", "Author", "BK_OT_"+suffix, 1)

	// Create transactions
	_, err := transactionService.BorrowBook(ctx, student.ID, targetBook.ID, librarian.ID, "Target book")
	require.NoError(t, err)

	// Return the target book so student can borrow another
	transactions, err := transactionService.GetTransactionHistory(ctx, student.ID, 10, 0)
	require.NoError(t, err)
	require.Len(t, transactions, 1)
	_, err = transactionService.ReturnBook(ctx, transactions[0].ID)
	require.NoError(t, err)

	_, err = transactionService.BorrowBook(ctx, student.ID, otherBook.ID, librarian.ID, "Other book")
	require.NoError(t, err)

	// Test: Filter by target book
	t.Run("FilterByBookID", func(t *testing.T) {
		result, err := transactionService.SearchTransactions(ctx, services.TransactionSearchParams{
			BookID: &targetBook.ID,
			Page:   1,
			Limit:  20,
		})
		require.NoError(t, err)

		// All results should be for the target book
		for _, tx := range result.Transactions {
			assert.Equal(t, targetBook.ID, tx.BookID)
		}
	})
}

func TestSearchTransactions_Combined(t *testing.T) {
	// Set up test database
	db := setupTestDB(t)
	defer db.Close()

	querier := queries.New(db)
	transactionService := services.NewTransactionService(querier)
	ctx := context.Background()

	// Create test data
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)
	student := createTestStudent(t, querier, "Combined", "Search", "STU_CB_"+suffix)
	librarian := createTestLibrarian(t, querier, "combined_librarian_"+suffix, "combined.lib."+suffix+"@example.com")
	book := createTestBook(t, querier, "Combined Test Book", "Combined Author", "BK_CB_"+suffix, 1)

	// Create a transaction
	_, err := transactionService.BorrowBook(ctx, student.ID, book.ID, librarian.ID, "Combined test")
	require.NoError(t, err)

	// Test: Combined search with multiple filters
	t.Run("CombinedFilters", func(t *testing.T) {
		now := time.Now()
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		endOfDay := startOfDay.Add(24 * time.Hour)

		result, err := transactionService.SearchTransactions(ctx, services.TransactionSearchParams{
			Query:     "Combined",
			StudentID: &student.ID,
			BookID:    &book.ID,
			Type:      "borrow",
			Status:    "active",
			FromDate:  &startOfDay,
			ToDate:    &endOfDay,
			Page:      1,
			Limit:     20,
		})
		require.NoError(t, err)

		// Should find our specific transaction
		assert.GreaterOrEqual(t, len(result.Transactions), 1)

		found := false
		for _, tx := range result.Transactions {
			if tx.StudentID == student.ID && tx.BookID == book.ID {
				found = true
				assert.Equal(t, "borrow", tx.TransactionType)
				assert.Equal(t, "active", tx.Status)
				break
			}
		}
		assert.True(t, found, "Expected to find the specific transaction")
	})
}

func TestSearchTransactions_Pagination(t *testing.T) {
	// Set up test database
	db := setupTestDB(t)
	defer db.Close()

	querier := queries.New(db)
	transactionService := services.NewTransactionService(querier)
	ctx := context.Background()

	// Create test data - multiple transactions
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)
	student := createTestStudent(t, querier, "Pagination", "Test", "STU_PG_"+suffix)
	librarian := createTestLibrarian(t, querier, "pagination_librarian_"+suffix, "pagination.lib."+suffix+"@example.com")

	// Create 5 books and borrow them
	for i := 0; i < 5; i++ {
		book := createTestBook(t, querier, fmt.Sprintf("Pagination Book %d", i), "Author", fmt.Sprintf("BK_PG_%s_%d", suffix, i), 1)
		_, err := transactionService.BorrowBook(ctx, student.ID, book.ID, librarian.ID, fmt.Sprintf("Pagination test %d", i))
		require.NoError(t, err)
	}

	// Test: Pagination with limit 2
	t.Run("PaginationLimit2", func(t *testing.T) {
		result, err := transactionService.SearchTransactions(ctx, services.TransactionSearchParams{
			StudentID: &student.ID,
			Page:      1,
			Limit:     2,
		})
		require.NoError(t, err)
		assert.Len(t, result.Transactions, 2)
		assert.Equal(t, 1, result.Pagination.Page)
		assert.Equal(t, 2, result.Pagination.Limit)
	})

	// Test: Get second page
	t.Run("SecondPage", func(t *testing.T) {
		result, err := transactionService.SearchTransactions(ctx, services.TransactionSearchParams{
			StudentID: &student.ID,
			Page:      2,
			Limit:     2,
		})
		require.NoError(t, err)
		assert.Len(t, result.Transactions, 2)
		assert.Equal(t, 2, result.Pagination.Page)
	})

	// Test: Get last page
	t.Run("LastPage", func(t *testing.T) {
		result, err := transactionService.SearchTransactions(ctx, services.TransactionSearchParams{
			StudentID: &student.ID,
			Page:      3,
			Limit:     2,
		})
		require.NoError(t, err)
		assert.Len(t, result.Transactions, 1) // Only 1 left on last page
		assert.Equal(t, 3, result.Pagination.Page)
	})
}

func TestSearchTransactions_Sorting(t *testing.T) {
	// Set up test database
	db := setupTestDB(t)
	defer db.Close()

	querier := queries.New(db)
	transactionService := services.NewTransactionService(querier)
	ctx := context.Background()

	// Create test data
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)
	student := createTestStudent(t, querier, "Sort", "Test", "STU_SO_"+suffix)
	librarian := createTestLibrarian(t, querier, "sort_librarian_"+suffix, "sort.lib."+suffix+"@example.com")

	// Create multiple transactions with slight time differences
	book1 := createTestBook(t, querier, "Sort Book 1", "Author", fmt.Sprintf("BK_SO_%s_1", suffix), 1)
	_, err := transactionService.BorrowBook(ctx, student.ID, book1.ID, librarian.ID, "Sort test 1")
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond) // Small delay

	book2 := createTestBook(t, querier, "Sort Book 2", "Author", fmt.Sprintf("BK_SO_%s_2", suffix), 1)
	_, err = transactionService.BorrowBook(ctx, student.ID, book2.ID, librarian.ID, "Sort test 2")
	require.NoError(t, err)

	// Test: Sort by transaction_date descending (default)
	t.Run("SortByDateDesc", func(t *testing.T) {
		result, err := transactionService.SearchTransactions(ctx, services.TransactionSearchParams{
			StudentID: &student.ID,
			SortBy:    "transaction_date",
			SortOrder: "desc",
			Page:      1,
			Limit:     10,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result.Transactions), 2)

		// Results should be in descending order
		if len(result.Transactions) >= 2 {
			assert.True(t, result.Transactions[0].TransactionDate.After(result.Transactions[1].TransactionDate) ||
				result.Transactions[0].TransactionDate.Equal(result.Transactions[1].TransactionDate))
		}
	})

	// Test: Sort by transaction_date ascending
	t.Run("SortByDateAsc", func(t *testing.T) {
		result, err := transactionService.SearchTransactions(ctx, services.TransactionSearchParams{
			StudentID: &student.ID,
			SortBy:    "transaction_date",
			SortOrder: "asc",
			Page:      1,
			Limit:     10,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result.Transactions), 2)

		// Results should be in ascending order
		if len(result.Transactions) >= 2 {
			assert.True(t, result.Transactions[0].TransactionDate.Before(result.Transactions[1].TransactionDate) ||
				result.Transactions[0].TransactionDate.Equal(result.Transactions[1].TransactionDate))
		}
	})
}

func TestSearchTransactions_TypeFilter(t *testing.T) {
	// Set up test database
	db := setupTestDB(t)
	defer db.Close()

	querier := queries.New(db)
	transactionService := services.NewTransactionService(querier)
	ctx := context.Background()

	// Create test data
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)
	student := createTestStudent(t, querier, "Type", "Filter", "STU_TF_"+suffix)
	librarian := createTestLibrarian(t, querier, "type_librarian_"+suffix, "type.lib."+suffix+"@example.com")
	book := createTestBook(t, querier, "Type Test Book", "Author", "BK_TF_"+suffix, 1)

	// Create a borrow transaction
	tx, err := transactionService.BorrowBook(ctx, student.ID, book.ID, librarian.ID, "Type test")
	require.NoError(t, err)

	// Renew the book (updates the existing transaction in place now)
	_, err = transactionService.RenewBook(ctx, tx.ID, librarian.ID, nil)
	require.NoError(t, err)

	// Test: Filter by borrow type
	t.Run("FilterByBorrowType", func(t *testing.T) {
		result, err := transactionService.SearchTransactions(ctx, services.TransactionSearchParams{
			StudentID: &student.ID,
			Type:      "borrow",
			Page:      1,
			Limit:     20,
		})
		require.NoError(t, err)

		for _, tx := range result.Transactions {
			assert.Equal(t, "borrow", tx.TransactionType)
		}
	})

	// Test: Filter by renew type
	t.Run("FilterByRenewType", func(t *testing.T) {
		result, err := transactionService.SearchTransactions(ctx, services.TransactionSearchParams{
			StudentID: &student.ID,
			Type:      "renew",
			Page:      1,
			Limit:     20,
		})
		require.NoError(t, err)

		for _, tx := range result.Transactions {
			assert.Equal(t, "renew", tx.TransactionType)
		}
	})
}

func TestSearchTransactions_OverdueCalculation(t *testing.T) {
	// Set up test database
	db := setupTestDB(t)
	defer db.Close()

	querier := queries.New(db)
	transactionService := services.NewTransactionService(querier)
	ctx := context.Background()

	// Create test data
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)
	student := createTestStudent(t, querier, "Overdue", "Calc", "STU_OC_"+suffix)
	librarian := createTestLibrarian(t, querier, "overdue_librarian_"+suffix, "overdue.lib."+suffix+"@example.com")
	book := createTestBook(t, querier, "Overdue Test Book", "Author", "BK_OC_"+suffix, 1)

	// Create a transaction
	createdTx, err := transactionService.BorrowBook(ctx, student.ID, book.ID, librarian.ID, "Overdue test")
	require.NoError(t, err)

	// Manually update the due date to be in the past
	pastDueDate := time.Now().Add(-5 * 24 * time.Hour) // 5 days ago
	_, err = db.Exec(ctx, "UPDATE transactions SET due_date = $1 WHERE id = $2", pastDueDate, createdTx.ID)
	require.NoError(t, err)

	// Search and check overdue calculation
	t.Run("OverdueStatusCalculation", func(t *testing.T) {
		result, err := transactionService.SearchTransactions(ctx, services.TransactionSearchParams{
			StudentID: &student.ID,
			Status:    "overdue",
			Page:      1,
			Limit:     20,
		})
		require.NoError(t, err)

		found := false
		for _, tx := range result.Transactions {
			if tx.ID == createdTx.ID && tx.StudentID == student.ID {
				found = true
				assert.Equal(t, "overdue", tx.Status)
				assert.GreaterOrEqual(t, tx.DaysOverdue, 4) // At least 4-5 days overdue
				break
			}
		}
		assert.True(t, found, "Expected to find the overdue transaction")
	})
}
