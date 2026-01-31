package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
)

// LanguageServiceInterface defines the language service methods
type LanguageServiceInterface interface {
	CreateLanguage(ctx context.Context, req models.CreateLanguageRequest) (*models.Language, error)
	GetLanguage(ctx context.Context, id int32) (*models.Language, error)
	GetLanguageByCode(ctx context.Context, code string) (*models.Language, error)
	ListLanguages(ctx context.Context, includeInactive bool, page, limit int) (*models.LanguageListResponse, error)
	SearchLanguages(ctx context.Context, query string, includeInactive bool, page, limit int) (*models.LanguageListResponse, error)
	UpdateLanguage(ctx context.Context, id int32, req models.UpdateLanguageRequest) (*models.Language, error)
	DeleteLanguage(ctx context.Context, id int32) error
	ActivateLanguage(ctx context.Context, id int32) (*models.Language, error)
	DeactivateLanguage(ctx context.Context, id int32) (*models.Language, error)
}

// LanguageService handles language operations
type LanguageService struct {
	queries queries.Querier
}

// NewLanguageService creates a new language service
func NewLanguageService(q queries.Querier) *LanguageService {
	return &LanguageService{queries: q}
}

func languageToModel(l queries.Language) *models.Language {
	var nativeName *string
	if l.NativeName.Valid {
		nativeName = &l.NativeName.String
	}
	return &models.Language{
		ID:         l.ID,
		Code:       l.Code,
		Name:       l.Name,
		NativeName: nativeName,
		IsActive:   l.IsActive.Bool,
		CreatedAt:  l.CreatedAt.Time,
		UpdatedAt:  l.UpdatedAt.Time,
	}
}

// CreateLanguage creates a new language
func (s *LanguageService) CreateLanguage(ctx context.Context, req models.CreateLanguageRequest) (*models.Language, error) {
	if req.Code == "" {
		return nil, errors.New("language code is required")
	}
	if req.Name == "" {
		return nil, errors.New("language name is required")
	}

	// Check if code already exists
	existing, _ := s.queries.GetLanguageByCode(ctx, req.Code)
	if existing.ID != 0 {
		return nil, errors.New("language code already exists")
	}

	var nativeName pgtype.Text
	if req.NativeName != nil {
		nativeName = pgtype.Text{String: *req.NativeName, Valid: true}
	}

	lang, err := s.queries.CreateLanguage(ctx, queries.CreateLanguageParams{
		Code:       req.Code,
		Name:       req.Name,
		NativeName: nativeName,
	})
	if err != nil {
		return nil, err
	}

	return languageToModel(lang), nil
}

// GetLanguage retrieves a language by ID
func (s *LanguageService) GetLanguage(ctx context.Context, id int32) (*models.Language, error) {
	lang, err := s.queries.GetLanguageByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return languageToModel(lang), nil
}

// GetLanguageByCode retrieves a language by code
func (s *LanguageService) GetLanguageByCode(ctx context.Context, code string) (*models.Language, error) {
	lang, err := s.queries.GetLanguageByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return languageToModel(lang), nil
}

// ListLanguages lists languages with pagination
func (s *LanguageService) ListLanguages(ctx context.Context, includeInactive bool, page, limit int) (*models.LanguageListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	// The generated code expects a bool for the filter
	// When includeInactive is true, we pass false to indicate no filter (NULL-like behavior)
	// When includeInactive is false, we pass true to filter only active languages
	isActiveFilter := !includeInactive

	langs, err := s.queries.ListLanguages(ctx, queries.ListLanguagesParams{
		Column1: isActiveFilter,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		return nil, err
	}

	count, err := s.queries.CountLanguages(ctx, isActiveFilter)
	if err != nil {
		return nil, err
	}

	languages := make([]models.Language, len(langs))
	for i, l := range langs {
		languages[i] = *languageToModel(l)
	}

	totalPages := int(count) / limit
	if int(count)%limit > 0 {
		totalPages++
	}

	return &models.LanguageListResponse{
		Languages: languages,
		Pagination: models.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      count,
			TotalPages: totalPages,
		},
	}, nil
}

// SearchLanguages searches languages by name or code
func (s *LanguageService) SearchLanguages(ctx context.Context, query string, includeInactive bool, page, limit int) (*models.LanguageListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	isActiveFilter := !includeInactive
	searchPattern := "%" + query + "%"

	langs, err := s.queries.SearchLanguages(ctx, queries.SearchLanguagesParams{
		Name:    searchPattern,
		Column2: isActiveFilter,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		return nil, err
	}

	count, err := s.queries.CountSearchLanguages(ctx, queries.CountSearchLanguagesParams{
		Name:    searchPattern,
		Column2: isActiveFilter,
	})
	if err != nil {
		return nil, err
	}

	languages := make([]models.Language, len(langs))
	for i, l := range langs {
		languages[i] = *languageToModel(l)
	}

	totalPages := int(count) / limit
	if int(count)%limit > 0 {
		totalPages++
	}

	return &models.LanguageListResponse{
		Languages: languages,
		Pagination: models.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      count,
			TotalPages: totalPages,
		},
	}, nil
}

// UpdateLanguage updates an existing language
func (s *LanguageService) UpdateLanguage(ctx context.Context, id int32, req models.UpdateLanguageRequest) (*models.Language, error) {
	// Check if language exists
	existing, err := s.queries.GetLanguageByID(ctx, id)
	if err != nil {
		return nil, errors.New("language not found")
	}

	// If code is being updated, check for duplicates
	if req.Code != nil && *req.Code != "" && *req.Code != existing.Code {
		existingByCode, _ := s.queries.GetLanguageByCode(ctx, *req.Code)
		if existingByCode.ID != 0 && existingByCode.ID != id {
			return nil, errors.New("language code already exists")
		}
	}

	// Build update params - use existing values if not provided
	params := queries.UpdateLanguageParams{
		ID:   id,
		Code: existing.Code,
		Name: existing.Name,
	}

	if req.Code != nil && *req.Code != "" {
		params.Code = *req.Code
	}
	if req.Name != nil && *req.Name != "" {
		params.Name = *req.Name
	}
	if req.NativeName != nil {
		params.NativeName = pgtype.Text{String: *req.NativeName, Valid: true}
	}
	if req.IsActive != nil {
		params.IsActive = pgtype.Bool{Bool: *req.IsActive, Valid: true}
	}

	lang, err := s.queries.UpdateLanguage(ctx, params)
	if err != nil {
		return nil, err
	}

	return languageToModel(lang), nil
}

// DeleteLanguage deletes a language
func (s *LanguageService) DeleteLanguage(ctx context.Context, id int32) error {
	// Check if language exists
	_, err := s.queries.GetLanguageByID(ctx, id)
	if err != nil {
		return errors.New("language not found")
	}

	return s.queries.DeleteLanguage(ctx, id)
}

// ActivateLanguage activates a language
func (s *LanguageService) ActivateLanguage(ctx context.Context, id int32) (*models.Language, error) {
	lang, err := s.queries.ActivateLanguage(ctx, id)
	if err != nil {
		return nil, err
	}
	return languageToModel(lang), nil
}

// DeactivateLanguage deactivates a language
func (s *LanguageService) DeactivateLanguage(ctx context.Context, id int32) (*models.Language, error) {
	lang, err := s.queries.DeactivateLanguage(ctx, id)
	if err != nil {
		return nil, err
	}
	return languageToModel(lang), nil
}
