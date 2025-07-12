package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
)

// ImportExportHandler handles import/export operations
type ImportExportHandler struct {
	importExportService services.ImportExportServiceInterface
}

// NewImportExportHandler creates a new import/export handler
func NewImportExportHandler(importExportService services.ImportExportServiceInterface) *ImportExportHandler {
	return &ImportExportHandler{
		importExportService: importExportService,
	}
}

// ImportBooks handles book import from CSV or Excel files
// @Summary Import books from file
// @Description Import books from CSV or Excel file
// @Tags import-export
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "CSV or Excel file"
// @Success 200 {object} models.ImportResult
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/import [post]
func (h *ImportExportHandler) ImportBooks(c *gin.Context) {
	// Get the uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "FILE_UPLOAD_ERROR",
				Message: "Failed to get uploaded file",
				Details: err.Error(),
			},
		})
		return
	}
	defer file.Close()

	// Validate file type
	fileName := header.Filename
	fileExt := strings.ToLower(filepath.Ext(fileName))
	
	if fileExt != ".csv" && fileExt != ".xlsx" && fileExt != ".xls" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNSUPPORTED_FORMAT",
				Message: "Only CSV and Excel files are supported",
				Details: "Supported formats: .csv, .xlsx, .xls",
			},
		})
		return
	}

	// Check for empty file
	if header.Size == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "EMPTY_FILE",
				Message: "Empty file uploaded",
				Details: "File size cannot be zero",
			},
		})
		return
	}

	// Check file size (10MB limit)
	if header.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "FILE_TOO_LARGE",
				Message: "File size exceeds 10MB limit",
			},
		})
		return
	}

	// Process the import based on file type
	var result *models.ImportResult
	if fileExt == ".csv" {
		result, err = h.importExportService.ImportBooksFromCSV(c.Request.Context(), file, fileName)
	} else {
		result, err = h.importExportService.ImportBooksFromExcel(c.Request.Context(), file, fileName)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "IMPORT_FAILED",
				Message: "Failed to import books",
				Details: err.Error(),
			},
		})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    result,
		Message: "Books imported successfully",
	})
}

// ExportBooks handles book export to CSV or Excel
// @Summary Export books to file
// @Description Export books to CSV or Excel file
// @Tags import-export
// @Accept json
// @Produce json
// @Param export_request body models.ExportRequest true "Export parameters"
// @Success 200 {object} models.ExportResult
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/export [post]
func (h *ImportExportHandler) ExportBooks(c *gin.Context) {
	var req models.ExportRequest
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

	// Validate format
	if req.Format != "csv" && req.Format != "excel" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid export format",
				Details: "Supported formats: csv, excel",
			},
		})
		return
	}

	// Process the export based on format
	if req.Format == "csv" {
		// Get CSV content directly
		content, fileName, err := h.importExportService.ExportBooksToCSVContent(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "EXPORT_FAILED",
					Message: "Failed to export books",
					Details: err.Error(),
				},
			})
			return
		}

		// Set headers for file download
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=\""+fileName+"\"")
		c.Header("Content-Transfer-Encoding", "binary")
		c.Header("Cache-Control", "no-cache")

		// Return CSV content
		c.String(http.StatusOK, content)
	} else {
		// For Excel, get content directly
		content, fileName, err := h.importExportService.ExportBooksToExcelContent(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "EXPORT_FAILED",
					Message: "Failed to export books",
					Details: err.Error(),
				},
			})
			return
		}

		// Set headers for Excel file download
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Header("Content-Disposition", "attachment; filename=\""+fileName+"\"")
		c.Header("Content-Transfer-Encoding", "binary")
		c.Header("Cache-Control", "no-cache")

		// Return Excel content
		c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", content)
	}
}

// GetImportTemplate generates and returns an import template
// @Summary Get import template
// @Description Get a template file for importing books
// @Tags import-export
// @Accept json
// @Produce json
// @Param format query string false "Template format (csv or excel)" default(csv)
// @Success 200 {object} models.ImportTemplate
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/import-template [get]
func (h *ImportExportHandler) GetImportTemplate(c *gin.Context) {
	format := c.DefaultQuery("format", "csv")
	
	if format != "csv" && format != "excel" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_FORMAT",
				Message: "Invalid template format",
				Details: "Supported formats: csv, excel",
			},
		})
		return
	}

	template, err := h.importExportService.GenerateImportTemplate(format)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "TEMPLATE_GENERATION_FAILED",
				Message: "Failed to generate import template",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    template,
		Message: "Import template generated successfully",
	})
}

// DownloadImportTemplate downloads the import template file
// @Summary Download import template file
// @Description Download a template file for importing books
// @Tags import-export
// @Accept json
// @Produce application/octet-stream
// @Param format query string false "Template format (csv or excel)" default(csv)
// @Success 200 {file} binary
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/import-template/download [get]
func (h *ImportExportHandler) DownloadImportTemplate(c *gin.Context) {
	format := c.DefaultQuery("format", "csv")
	
	if format != "csv" && format != "excel" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_FORMAT",
				Message: "Invalid template format",
				Details: "Supported formats: csv, excel",
			},
		})
		return
	}

	template, err := h.importExportService.GenerateImportTemplate(format)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "TEMPLATE_GENERATION_FAILED",
				Message: "Failed to generate import template",
				Details: err.Error(),
			},
		})
		return
	}

	// Set response headers
	fileName := "book_import_template." + format
	if format == "excel" {
		fileName = "book_import_template.xlsx"
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	} else {
		c.Header("Content-Type", "text/csv")
	}
	
	c.Header("Content-Disposition", "attachment; filename=\""+fileName+"\"")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Cache-Control", "no-cache")

	// Generate file content based on format
	if format == "csv" {
		// Generate CSV content with headers and sample data
		csvContent := "book_id,title,author,isbn,publisher,published_year,genre,description,total_copies,available_copies,shelf_location\n"
		for _, sample := range template.SampleData {
			// Helper function to safely get string value
			safeString := func(ptr *string) string {
				if ptr == nil {
					return ""
				}
				return *ptr
			}
			
			// Helper function to safely get int32 value
			safeInt32 := func(ptr *int32) int32 {
				if ptr == nil {
					return 0
				}
				return *ptr
			}
			
			csvContent += fmt.Sprintf("%s,%s,%s,%s,%s,%d,%s,%s,%d,%d,%s\n",
				sample.BookID,
				sample.Title,
				sample.Author,
				safeString(sample.ISBN),
				safeString(sample.Publisher),
				safeInt32(sample.PublishedYear),
				safeString(sample.Genre),
				safeString(sample.Description),
				safeInt32(sample.TotalCopies),
				safeInt32(sample.AvailableCopies),
				safeString(sample.ShelfLocation),
			)
		}
		c.String(http.StatusOK, csvContent)
	} else {
		// For now, return JSON for Excel (would need Excel generation library)
		c.JSON(http.StatusOK, template)
	}
}

// GetImportHistory gets the history of import operations
// @Summary Get import history
// @Description Get history of book import operations
// @Tags import-export
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/import-history [get]
func (h *ImportExportHandler) GetImportHistory(c *gin.Context) {
	page, limit := parsePaginationParams(c)

	// TODO: Implement import history storage and retrieval
	// For now, return placeholder data
	history := map[string]interface{}{
		"imports": []interface{}{},
		"pagination": map[string]interface{}{
			"page":  page,
			"limit": limit,
			"total": 0,
		},
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    history,
		Message: "Import history retrieved successfully",
	})
}

// GetExportHistory gets the history of export operations
// @Summary Get export history
// @Description Get history of book export operations
// @Tags import-export
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/books/export-history [get]
func (h *ImportExportHandler) GetExportHistory(c *gin.Context) {
	page, limit := parsePaginationParams(c)

	// TODO: Implement export history storage and retrieval
	// For now, return placeholder data
	history := map[string]interface{}{
		"exports": []interface{}{},
		"pagination": map[string]interface{}{
			"page":  page,
			"limit": limit,
			"total": 0,
		},
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    history,
		Message: "Export history retrieved successfully",
	})
}