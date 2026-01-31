package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
)

// BookHandler handles book-related HTTP requests
type BookHandler struct {
	bookService           services.BookServiceInterface
	isbnService           services.ISBNServiceInterface
	recommendationService services.RecommendationServiceInterface
}

// NewBookHandler creates a new book handler
func NewBookHandler(bookService services.BookServiceInterface, isbnService services.ISBNServiceInterface, recommendationService services.RecommendationServiceInterface) *BookHandler {
	return &BookHandler{
		bookService:           bookService,
		isbnService:           isbnService,
		recommendationService: recommendationService,
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
		slog.Error("Failed to create book", "error", err)
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

// FetchBookByISBN fetches book information from external APIs using ISBN
// @Summary Fetch book info by ISBN
// @Description Fetch book information from external APIs (Google Books, Open Library) using ISBN
// @Tags books
// @Accept json
// @Produce json
// @Param isbn body models.ISBNFetchRequest true "ISBN to fetch"
// @Success 200 {object} models.ISBNBookInfo
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/isbn/fetch [post]
func (h *BookHandler) FetchBookByISBN(c *gin.Context) {
	var req models.ISBNFetchRequest
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

	bookInfo, err := h.isbnService.FetchBookInfoByISBN(c.Request.Context(), req.ISBN)
	if err != nil {
		if strings.Contains(err.Error(), "invalid ISBN") {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: err.Error(),
				},
			})
			return
		}
		if strings.Contains(err.Error(), "no book information found") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Book information not found for the provided ISBN",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to fetch book information",
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    bookInfo,
		Message: "Book information fetched successfully",
	})
}

// ProcessBarcode processes a scanned barcode and returns book information
// @Summary Process barcode scan
// @Description Process a scanned barcode (ISBN, EAN, UPC) and return book information
// @Tags books
// @Accept json
// @Produce json
// @Param barcode body models.BarcodeScanRequest true "Barcode data"
// @Success 200 {object} models.ISBNBookInfo
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/barcode/scan [post]
func (h *BookHandler) ProcessBarcode(c *gin.Context) {
	var req models.BarcodeScanRequest
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

	// For now, we'll treat all barcode types as ISBN
	// In the future, we can add specific handling for EAN and UPC codes
	var isbn string
	switch strings.ToLower(req.Type) {
	case "isbn":
		isbn = req.Barcode
	case "ean":
		// EAN-13 can be treated as ISBN-13 for books
		if len(req.Barcode) == 13 && (strings.HasPrefix(req.Barcode, "978") || strings.HasPrefix(req.Barcode, "979")) {
			isbn = req.Barcode
		} else {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: "EAN code does not appear to be a book ISBN",
				},
			})
			return
		}
	case "upc":
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "UPC codes are not supported for book lookup",
			},
		})
		return
	default:
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Unsupported barcode type",
			},
		})
		return
	}

	bookInfo, err := h.isbnService.FetchBookInfoByISBN(c.Request.Context(), isbn)
	if err != nil {
		if strings.Contains(err.Error(), "invalid ISBN") {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: err.Error(),
				},
			})
			return
		}
		if strings.Contains(err.Error(), "no book information found") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Book information not found for the scanned barcode",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to process barcode",
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    bookInfo,
		Message: "Barcode processed successfully",
	})
}

// ProcessRichTextDescription processes rich text description for a book
// @Summary Process rich text description
// @Description Process and sanitize rich text description for security and formatting
// @Tags books
// @Accept json
// @Produce json
// @Param description body models.RichTextDescriptionRequest true "Rich text description data"
// @Success 200 {object} models.RichTextContent
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/description/process [post]
func (h *BookHandler) ProcessRichTextDescription(c *gin.Context) {
	var req models.RichTextDescriptionRequest
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

	processedContent, err := h.bookService.ProcessRichTextDescription(c.Request.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Book not found",
				},
			})
			return
		}
		if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "required") {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to process rich text description",
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    processedContent,
		Message: "Rich text description processed successfully",
	})
}

// GetRecommendations gets book recommendations based on various strategies
// @Summary Get book recommendations
// @Description Get personalized book recommendations using various strategies
// @Tags books
// @Accept json
// @Produce json
// @Param student_id query int false "Student ID for personalized recommendations"
// @Param genre query string false "Genre filter"
// @Param author query string false "Author filter"
// @Param timeframe query string false "Timeframe for popular books (week/month/year/all)" default(month)
// @Param limit query int false "Number of recommendations (1-50)" default(10)
// @Param strategy query string false "Recommendation strategy (auto/personalized/genre/popularity/similarity/recent)" default(auto)
// @Success 200 {object} models.BookRecommendationsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/recommendations [get]
func (h *BookHandler) GetRecommendations(c *gin.Context) {
	var req models.RecommendationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid query parameters",
				Details: err.Error(),
			},
		})
		return
	}

	// Set defaults
	if req.Limit < 1 || req.Limit > 50 {
		req.Limit = 10
	}
	if req.Timeframe == "" {
		req.Timeframe = "month"
	}
	if req.Strategy == "" {
		req.Strategy = "auto"
	}

	var recommendations *models.BookRecommendationsResponse
	var err error

	// Route to appropriate recommendation strategy
	switch req.Strategy {
	case "personalized":
		if req.StudentID == nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: "student_id is required for personalized recommendations",
				},
			})
			return
		}
		recommendations, err = h.recommendationService.GetRecommendationsForStudent(c.Request.Context(), *req.StudentID, req.Limit)

	case "genre":
		if req.Genre == nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: "genre is required for genre-based recommendations",
				},
			})
			return
		}
		recommendations, err = h.recommendationService.GetRecommendationsByGenre(c.Request.Context(), *req.Genre, req.Limit)

	case "popularity":
		recommendations, err = h.recommendationService.GetPopularBooks(c.Request.Context(), req.Timeframe, req.Limit)

	case "recent":
		recommendations, err = h.recommendationService.GetRecentAdditions(c.Request.Context(), req.Limit)

	case "auto":
		// Auto strategy: use personalized if student_id provided, otherwise popular
		if req.StudentID != nil {
			recommendations, err = h.recommendationService.GetRecommendationsForStudent(c.Request.Context(), *req.StudentID, req.Limit)
		} else if req.Genre != nil {
			recommendations, err = h.recommendationService.GetRecommendationsByGenre(c.Request.Context(), *req.Genre, req.Limit)
		} else {
			recommendations, err = h.recommendationService.GetPopularBooks(c.Request.Context(), req.Timeframe, req.Limit)
		}

	default:
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid recommendation strategy",
			},
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to get recommendations",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    recommendations,
		Message: "Recommendations retrieved successfully",
	})
}

// GetSimilarBooks gets books similar to a specific book
// @Summary Get similar books
// @Description Get books similar to a specific book based on author and genre
// @Tags books
// @Accept json
// @Produce json
// @Param id path int true "Book ID"
// @Param limit query int false "Number of similar books (1-50)" default(10)
// @Success 200 {object} models.BookRecommendationsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/{id}/similar [get]
func (h *BookHandler) GetSimilarBooks(c *gin.Context) {
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

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 50 {
		limit = 10
	}

	recommendations, err := h.recommendationService.GetSimilarBooks(c.Request.Context(), int32(id), limit)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
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
				Message: "Failed to get similar books",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    recommendations,
		Message: "Similar books retrieved successfully",
	})
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
