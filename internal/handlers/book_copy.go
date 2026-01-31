package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
)

// BookCopyHandler handles book copy-related HTTP requests
type BookCopyHandler struct {
	bookCopyService services.BookCopyServiceInterface
}

// NewBookCopyHandler creates a new book copy handler
func NewBookCopyHandler(bookCopyService services.BookCopyServiceInterface) *BookCopyHandler {
	return &BookCopyHandler{
		bookCopyService: bookCopyService,
	}
}

// CreateBookCopy creates a new book copy
// @Summary Create a new book copy
// @Description Create a new copy of a book
// @Tags book-copies
// @Accept json
// @Produce json
// @Param id path int true "Book ID"
// @Param copy body models.CreateBookCopyRequest true "Copy data"
// @Success 201 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/{id}/copies [post]
func (h *BookCopyHandler) CreateBookCopy(c *gin.Context) {
	idStr := c.Param("id")
	bookID, err := strconv.ParseInt(idStr, 10, 32)
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

	var req models.CreateBookCopyRequest
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

	req.BookID = int32(bookID)

	copy, err := h.bookCopyService.CreateBookCopy(c.Request.Context(), req)
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
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to create book copy",
			},
		})
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Data:    copy,
		Message: "Book copy created successfully",
	})
}

// GetBookCopy retrieves a specific book copy
// @Summary Get a book copy by ID
// @Description Retrieve a single book copy by its ID
// @Tags book-copies
// @Accept json
// @Produce json
// @Param id path int true "Book ID"
// @Param copy_id path int true "Copy ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/{id}/copies/{copy_id} [get]
func (h *BookCopyHandler) GetBookCopy(c *gin.Context) {
	copyIDStr := c.Param("copy_id")
	copyID, err := strconv.ParseInt(copyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid copy ID",
			},
		})
		return
	}

	copy, err := h.bookCopyService.GetBookCopyByID(c.Request.Context(), int32(copyID))
	if err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Book copy not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to get book copy",
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    copy,
	})
}

// ListBookCopies lists all copies of a book
// @Summary List all copies of a book
// @Description Get all copies of a specific book
// @Tags book-copies
// @Accept json
// @Produce json
// @Param id path int true "Book ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/{id}/copies [get]
func (h *BookCopyHandler) ListBookCopies(c *gin.Context) {
	idStr := c.Param("id")
	bookID, err := strconv.ParseInt(idStr, 10, 32)
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

	copies, err := h.bookCopyService.ListBookCopies(c.Request.Context(), int32(bookID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to list book copies",
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    copies,
	})
}

// UpdateBookCopy updates a book copy
// @Summary Update a book copy
// @Description Update an existing book copy
// @Tags book-copies
// @Accept json
// @Produce json
// @Param id path int true "Book ID"
// @Param copy_id path int true "Copy ID"
// @Param copy body models.UpdateBookCopyRequest true "Copy data"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/{id}/copies/{copy_id} [put]
func (h *BookCopyHandler) UpdateBookCopy(c *gin.Context) {
	copyIDStr := c.Param("copy_id")
	copyID, err := strconv.ParseInt(copyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid copy ID",
			},
		})
		return
	}

	var req models.UpdateBookCopyRequest
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

	copy, err := h.bookCopyService.UpdateBookCopy(c.Request.Context(), int32(copyID), req)
	if err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Book copy not found",
				},
			})
			return
		}
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
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to update book copy",
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    copy,
		Message: "Book copy updated successfully",
	})
}

// DeleteBookCopy deletes a book copy
// @Summary Delete a book copy
// @Description Delete a book copy
// @Tags book-copies
// @Accept json
// @Produce json
// @Param id path int true "Book ID"
// @Param copy_id path int true "Copy ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/{id}/copies/{copy_id} [delete]
func (h *BookCopyHandler) DeleteBookCopy(c *gin.Context) {
	copyIDStr := c.Param("copy_id")
	copyID, err := strconv.ParseInt(copyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid copy ID",
			},
		})
		return
	}

	err = h.bookCopyService.DeleteBookCopy(c.Request.Context(), int32(copyID))
	if err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Book copy not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to delete book copy",
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Book copy deleted successfully",
	})
}

// ScanBarcode looks up a book copy by barcode
// @Summary Scan barcode to find copy
// @Description Look up a book copy by its barcode
// @Tags book-copies
// @Accept json
// @Produce json
// @Param barcode query string true "Barcode"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/copies/scan [get]
func (h *BookCopyHandler) ScanBarcode(c *gin.Context) {
	barcode := c.Query("barcode")
	if barcode == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Barcode is required",
			},
		})
		return
	}

	copy, err := h.bookCopyService.GetBookCopyByBarcode(c.Request.Context(), barcode)
	if err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "No book copy found with this barcode",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to look up barcode",
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    copy,
	})
}
