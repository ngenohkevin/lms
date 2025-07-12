package handlers

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
)

// UploadHandler handles file upload operations
type UploadHandler struct {
	bookService *services.BookService
}

// NewUploadHandler creates a new upload handler
func NewUploadHandler(bookService *services.BookService) *UploadHandler {
	return &UploadHandler{
		bookService: bookService,
	}
}

// UploadBookCover handles book cover image upload
// @Summary Upload book cover image
// @Description Upload a cover image for a book
// @Tags books
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "Book ID"
// @Param cover formData file true "Cover image file"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 413 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/{id}/cover [post]
func (h *UploadHandler) UploadBookCover(c *gin.Context) {
	// Get book ID from URL parameter
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

	// Check if book exists
	book, err := h.bookService.GetBookByID(c.Request.Context(), int32(bookID))
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

	// Get the uploaded file
	file, header, err := c.Request.FormFile("cover")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "No file uploaded or invalid file",
			},
		})
		return
	}
	defer file.Close()

	// Validate file
	if err := validateImageFile(header); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	// Create uploads directory if it doesn't exist
	uploadDir := getUploadDir()
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to create upload directory",
			},
		})
		return
	}

	// Generate unique filename
	filename := generateUniqueFilename(header.Filename, int32(bookID))
	filePath := filepath.Join(uploadDir, filename)

	// Create the destination file
	dst, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to create destination file",
			},
		})
		return
	}
	defer dst.Close()

	// Copy the uploaded file to the destination
	if _, err := io.Copy(dst, file); err != nil {
		// Clean up the file if copy fails
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to save uploaded file",
			},
		})
		return
	}

	// Generate the public URL for the image
	imageURL := generateImageURL(filename)

	// Update the book with the new cover image URL
	updateReq := models.UpdateBookRequest{
		CoverImageURL: &imageURL,
	}

	updatedBook, err := h.bookService.UpdateBook(c.Request.Context(), int32(bookID), updateReq)
	if err != nil {
		// Clean up the uploaded file if database update fails
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to update book with cover image",
			},
		})
		return
	}

	// Clean up old cover image if it exists
	if book.CoverImageURL != nil && *book.CoverImageURL != "" {
		oldFilename := extractFilenameFromURL(*book.CoverImageURL)
		if oldFilename != "" {
			oldFilePath := filepath.Join(uploadDir, oldFilename)
			os.Remove(oldFilePath) // Ignore error if file doesn't exist
		}
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data: gin.H{
			"book":       updatedBook,
			"image_url":  imageURL,
			"filename":   filename,
			"file_size":  header.Size,
		},
		Message: "Book cover uploaded successfully",
	})
}

// DeleteBookCover removes a book's cover image
// @Summary Delete book cover image
// @Description Remove the cover image from a book
// @Tags books
// @Accept json
// @Produce json
// @Param id path int true "Book ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/{id}/cover [delete]
func (h *UploadHandler) DeleteBookCover(c *gin.Context) {
	// Get book ID from URL parameter
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

	// Check if book exists
	book, err := h.bookService.GetBookByID(c.Request.Context(), int32(bookID))
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

	// Check if book has a cover image
	if book.CoverImageURL == nil || *book.CoverImageURL == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Book has no cover image to delete",
			},
		})
		return
	}

	// Remove the cover image URL from the book
	emptyURL := ""
	updateReq := models.UpdateBookRequest{
		CoverImageURL: &emptyURL,
	}

	updatedBook, err := h.bookService.UpdateBook(c.Request.Context(), int32(bookID), updateReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to update book",
			},
		})
		return
	}

	// Delete the physical file
	filename := extractFilenameFromURL(*book.CoverImageURL)
	if filename != "" {
		uploadDir := getUploadDir()
		filePath := filepath.Join(uploadDir, filename)
		os.Remove(filePath) // Ignore error if file doesn't exist
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    updatedBook,
		Message: "Book cover deleted successfully",
	})
}

// Helper functions

// validateImageFile validates that the uploaded file is a valid image
func validateImageFile(header *multipart.FileHeader) error {
	const maxFileSize = 5 * 1024 * 1024 // 5MB

	// Check file size
	if header.Size > maxFileSize {
		return fmt.Errorf("file size exceeds maximum allowed size of %d bytes", maxFileSize)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowedExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	
	isAllowed := false
	for _, allowedExt := range allowedExtensions {
		if ext == allowedExt {
			isAllowed = true
			break
		}
	}
	
	if !isAllowed {
		return fmt.Errorf("file type not allowed. Allowed types: %v", allowedExtensions)
	}

	// Check MIME type
	allowedMimeTypes := []string{
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
	}
	
	mimeType := header.Header.Get("Content-Type")
	isAllowedMime := false
	for _, allowedMime := range allowedMimeTypes {
		if mimeType == allowedMime {
			isAllowedMime = true
			break
		}
	}
	
	if !isAllowedMime {
		return fmt.Errorf("invalid MIME type. Allowed types: %v", allowedMimeTypes)
	}

	return nil
}

// generateUniqueFilename generates a unique filename for the uploaded file
func generateUniqueFilename(originalFilename string, bookID int32) string {
	ext := filepath.Ext(originalFilename)
	timestamp := time.Now().Unix()
	return fmt.Sprintf("book_%d_cover_%d%s", bookID, timestamp, ext)
}

// getUploadDir returns the directory where uploaded files are stored
func getUploadDir() string {
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads/book-covers"
	}
	return uploadDir
}

// generateImageURL generates the public URL for the uploaded image
func generateImageURL(filename string) string {
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return fmt.Sprintf("%s/uploads/book-covers/%s", baseURL, filename)
}

// extractFilenameFromURL extracts the filename from an image URL
func extractFilenameFromURL(imageURL string) string {
	parts := strings.Split(imageURL, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}