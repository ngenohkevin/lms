package services

import (
	"context"
	"fmt"

	"github.com/skip2/go-qrcode"
)

// QRCodeServiceInterface defines the interface for QR code operations
type QRCodeServiceInterface interface {
	GenerateBookQR(ctx context.Context, bookID string, baseURL string) ([]byte, error)
	GenerateCopyQR(ctx context.Context, copyID int32, barcode string, baseURL string) ([]byte, error)
}

// QRCodeService handles QR code generation
type QRCodeService struct {
	size int // QR code size in pixels
}

// NewQRCodeService creates a new QR code service
func NewQRCodeService() *QRCodeService {
	return &QRCodeService{
		size: 256, // Default size
	}
}

// GenerateBookQR generates a QR code for a book
// The QR code contains a URL that links to the book detail page
func (s *QRCodeService) GenerateBookQR(ctx context.Context, bookID string, baseURL string) ([]byte, error) {
	// Generate URL for the book
	url := fmt.Sprintf("%s/books/%s", baseURL, bookID)

	// Generate QR code as PNG bytes
	png, err := qrcode.Encode(url, qrcode.Medium, s.size)
	if err != nil {
		return nil, fmt.Errorf("failed to generate QR code: %w", err)
	}

	return png, nil
}

// GenerateCopyQR generates a QR code for a specific book copy
// The QR code contains the barcode or copy ID for scanning
func (s *QRCodeService) GenerateCopyQR(ctx context.Context, copyID int32, barcode string, baseURL string) ([]byte, error) {
	// Use barcode if available, otherwise use copy ID
	var content string
	if barcode != "" {
		content = barcode
	} else {
		content = fmt.Sprintf("COPY-%d", copyID)
	}

	// For copies, we include both the scan code and a URL
	url := fmt.Sprintf("%s/copies/%d", baseURL, copyID)
	qrContent := fmt.Sprintf("%s\n%s", content, url)

	// Generate QR code as PNG bytes
	png, err := qrcode.Encode(qrContent, qrcode.Medium, s.size)
	if err != nil {
		return nil, fmt.Errorf("failed to generate QR code: %w", err)
	}

	return png, nil
}

// SetSize sets the QR code size in pixels
func (s *QRCodeService) SetSize(size int) {
	if size > 0 {
		s.size = size
	}
}
