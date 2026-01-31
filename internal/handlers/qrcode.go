package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/services"
)

// QRCodeHandler handles QR code generation endpoints
type QRCodeHandler struct {
	qrService   services.QRCodeServiceInterface
	bookService services.BookServiceInterface
	copyService services.BookCopyServiceInterface
	baseURL     string
}

// NewQRCodeHandler creates a new QR code handler
func NewQRCodeHandler(qrService services.QRCodeServiceInterface, bookService services.BookServiceInterface, copyService services.BookCopyServiceInterface, baseURL string) *QRCodeHandler {
	return &QRCodeHandler{
		qrService:   qrService,
		bookService: bookService,
		copyService: copyService,
		baseURL:     baseURL,
	}
}

// GetBookQR generates a QR code for a book
// @Summary Generate QR code for a book
// @Description Generate a QR code image containing a URL to the book detail page
// @Tags books
// @Produce image/png
// @Param id path int true "Book ID"
// @Success 200 {file} binary "QR code image"
// @Failure 400 {object} models.ErrorResponse "Invalid book ID"
// @Failure 404 {object} models.ErrorResponse "Book not found"
// @Failure 500 {object} models.ErrorResponse "Failed to generate QR code"
// @Router /api/v1/books/{id}/qr [get]
func (h *QRCodeHandler) GetBookQR(c *gin.Context) {
	// Parse book ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid book ID"})
		return
	}

	// Get the book to verify it exists
	book, err := h.bookService.GetBookByID(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}

	// Generate QR code
	png, err := h.qrService.GenerateBookQR(c.Request.Context(), book.BookID, h.baseURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate QR code"})
		return
	}

	// Set headers for PNG image download
	c.Header("Content-Type", "image/png")
	c.Header("Content-Disposition", "inline; filename=\"book-"+book.BookID+"-qr.png\"")
	c.Data(http.StatusOK, "image/png", png)
}

// GetCopyQR generates a QR code for a specific book copy
// @Summary Generate QR code for a book copy
// @Description Generate a QR code image containing the copy barcode/ID
// @Tags copies
// @Produce image/png
// @Param copy_id path int true "Copy ID"
// @Success 200 {file} binary "QR code image"
// @Failure 400 {object} models.ErrorResponse "Invalid copy ID"
// @Failure 404 {object} models.ErrorResponse "Copy not found"
// @Failure 500 {object} models.ErrorResponse "Failed to generate QR code"
// @Router /api/v1/books/copies/{copy_id}/qr [get]
func (h *QRCodeHandler) GetCopyQR(c *gin.Context) {
	// Parse copy ID
	copyIDStr := c.Param("copy_id")
	copyID, err := strconv.ParseInt(copyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid copy ID"})
		return
	}

	// Get the copy to verify it exists
	copy, err := h.copyService.GetBookCopyByID(c.Request.Context(), int32(copyID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "copy not found"})
		return
	}

	// Get barcode if available
	barcode := ""
	if copy.Barcode != nil {
		barcode = *copy.Barcode
	}

	// Generate QR code
	png, err := h.qrService.GenerateCopyQR(c.Request.Context(), int32(copyID), barcode, h.baseURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate QR code"})
		return
	}

	// Set headers for PNG image download
	c.Header("Content-Type", "image/png")
	c.Header("Content-Disposition", "inline; filename=\"copy-"+copyIDStr+"-qr.png\"")
	c.Data(http.StatusOK, "image/png", png)
}
