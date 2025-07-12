package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
)

// BookHandler handles book-related HTTP requests
type BookHandler struct {
	bookService services.BookServiceInterface
}

// NewBookHandler creates a new book handler
func NewBookHandler(bookService services.BookServiceInterface) *BookHandler {
	return &BookHandler{
		bookService: bookService,
	}
}

// CreateBook creates a new book
// @Summary Create a new book
// @Description Create a new book in the library system
// @Tags books
// @Accept json
// @Produce json
// @Param book body models.CreateBookRequest true "Book data"
// @Success 201 {object} models.BookResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books [post]
func (h *BookHandler) CreateBook(c *gin.Context) {
	var req models.CreateBookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request data",
				Details: err.Error(),
			},
		})
		return
	}

	book, err := h.bookService.CreateBook(c.Request.Context(), req)
	if err != nil {
		if isValidationError(err) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: err.Error(),
				},
			})
			return
		}
		if isConflictError(err) {
			c.JSON(http.StatusConflict, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "CONFLICT_ERROR",
					Message: err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to create book",
			},
		})
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Data:    book,
		Message: "Book created successfully",
	})
}

// GetBook retrieves a book by ID
// @Summary Get a book by ID
// @Description Retrieve a single book by its ID
// @Tags books
// @Accept json
// @Produce json
// @Param id path int true "Book ID"
// @Success 200 {object} models.BookResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/{id} [get]
func (h *BookHandler) GetBook(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid book ID",
			},
		})
		return
	}

	book, err := h.bookService.GetBookByID(c.Request.Context(), int32(id))
	if err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Book not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to retrieve book",
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    book,
	})
}

// GetBookByBookID retrieves a book by its custom book ID
// @Summary Get a book by BookID
// @Description Retrieve a single book by its custom book ID
// @Tags books
// @Accept json
// @Produce json
// @Param book_id path string true "Book ID"
// @Success 200 {object} models.BookResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/book/{book_id} [get]
func (h *BookHandler) GetBookByBookID(c *gin.Context) {
	bookID := c.Param("book_id")
	if bookID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Book ID is required",
			},
		})
		return
	}

	book, err := h.bookService.GetBookByBookID(c.Request.Context(), bookID)
	if err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Book not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to retrieve book",
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    book,
	})
}

// UpdateBook updates an existing book
// @Summary Update a book
// @Description Update an existing book's information
// @Tags books
// @Accept json
// @Produce json
// @Param id path int true "Book ID"
// @Param book body models.UpdateBookRequest true "Book data"
// @Success 200 {object} models.BookResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/{id} [put]
func (h *BookHandler) UpdateBook(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid book ID",
			},
		})
		return
	}

	var req models.UpdateBookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request data",
				Details: err.Error(),
			},
		})
		return
	}

	book, err := h.bookService.UpdateBook(c.Request.Context(), int32(id), req)
	if err != nil {
		if isValidationError(err) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: err.Error(),
				},
			})
			return
		}
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Book not found",
				},
			})
			return
		}
		if isConflictError(err) {
			c.JSON(http.StatusConflict, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "CONFLICT_ERROR",
					Message: err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to update book",
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    book,
		Message: "Book updated successfully",
	})
}

// DeleteBook soft deletes a book
// @Summary Delete a book
// @Description Soft delete a book from the library system
// @Tags books
// @Accept json
// @Produce json
// @Param id path int true "Book ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/{id} [delete]
func (h *BookHandler) DeleteBook(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid book ID",
			},
		})
		return
	}

	err = h.bookService.DeleteBook(c.Request.Context(), int32(id))
	if err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Book not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to delete book",
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Book deleted successfully",
	})
}

// ListBooks lists all books with pagination
// @Summary List books
// @Description Get a paginated list of all books
// @Tags books
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Success 200 {object} models.BookListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books [get]
func (h *BookHandler) ListBooks(c *gin.Context) {
	page, limit := parsePaginationParams(c)

	books, err := h.bookService.ListBooks(c.Request.Context(), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to list books",
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    books,
	})
}

// SearchBooks searches for books with various filters
// @Summary Search books
// @Description Search for books with various filters and pagination
// @Tags books
// @Accept json
// @Produce json
// @Param query query string false "Search query (title, author, ISBN, book_id)"
// @Param genre query string false "Genre filter"
// @Param author query string false "Author filter"
// @Param published_year query int false "Published year filter"
// @Param available_only query bool false "Show only available books"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Success 200 {object} models.BookListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/search [get]
func (h *BookHandler) SearchBooks(c *gin.Context) {
	var req models.BookSearchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid search parameters",
				Details: err.Error(),
			},
		})
		return
	}

	// Set default values
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	books, err := h.bookService.SearchBooks(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to search books",
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    books,
	})
}

// GetBookStats returns book statistics
// @Summary Get book statistics
// @Description Get statistics about books in the library
// @Tags books
// @Accept json
// @Produce json
// @Success 200 {object} models.BookStats
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/stats [get]
func (h *BookHandler) GetBookStats(c *gin.Context) {
	stats, err := h.bookService.GetBookStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to get book statistics",
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    stats,
	})
}

// Helper functions

func parsePaginationParams(c *gin.Context) (page, limit int) {
	page = 1
	limit = 20

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > 100 {
				limit = 100
			}
		}
	}

	return page, limit
}

func isValidationError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "validation error")
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "not found") ||
		strings.Contains(errMsg, "no rows in result set") ||
		strings.Contains(errMsg, "failed to get")
}

func isConflictError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "already exists") ||
		strings.Contains(errMsg, "duplicate")
}