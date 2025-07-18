package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/ngenohkevin/lms/internal/models"
)

// EmailTemplateManagerInterface defines the interface for email template management
type EmailTemplateManagerInterface interface {
	GetTemplate(ctx context.Context, name string) (*models.EmailTemplate, error)
	CreateTemplate(ctx context.Context, template *models.EmailTemplate) (*models.EmailTemplate, error)
	UpdateTemplate(ctx context.Context, template *models.EmailTemplate) (*models.EmailTemplate, error)
	DeleteTemplate(ctx context.Context, name string) error
	ListTemplates(ctx context.Context, filter *models.TemplateFilter) ([]*models.EmailTemplate, error)
	ValidateTemplate(ctx context.Context, template *models.EmailTemplate) error
	TestTemplate(ctx context.Context, templateName string, testData map[string]interface{}) (*models.TemplateTestResult, error)
	DuplicateTemplate(ctx context.Context, sourceName, newName string) (*models.EmailTemplate, error)
	GetTemplateVariables(ctx context.Context, templateName string) ([]string, error)
	BackupTemplates(ctx context.Context) ([]*models.EmailTemplate, error)
	RestoreTemplates(ctx context.Context, templates []*models.EmailTemplate) error
}

// EmailTemplateManager manages email templates
type EmailTemplateManager struct {
	templates map[string]*models.EmailTemplate
	mutex     sync.RWMutex
	logger    *slog.Logger
}

// NewEmailTemplateManager creates a new email template manager
func NewEmailTemplateManager(logger *slog.Logger) *EmailTemplateManager {
	manager := &EmailTemplateManager{
		templates: make(map[string]*models.EmailTemplate),
		logger:    logger,
	}

	// Load default templates
	manager.loadDefaultTemplates()

	return manager
}

// GetTemplate retrieves a template by name
func (m *EmailTemplateManager) GetTemplate(ctx context.Context, name string) (*models.EmailTemplate, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	template, exists := m.templates[name]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", name)
	}

	// Return a copy to prevent external modification
	return m.copyTemplate(template), nil
}

// CreateTemplate creates a new template
func (m *EmailTemplateManager) CreateTemplate(ctx context.Context, template *models.EmailTemplate) (*models.EmailTemplate, error) {
	if err := m.ValidateTemplate(ctx, template); err != nil {
		return nil, fmt.Errorf("template validation failed: %w", err)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if template already exists
	if _, exists := m.templates[template.Name]; exists {
		return nil, fmt.Errorf("template already exists: %s", template.Name)
	}

	// Set metadata
	now := time.Now()
	template.CreatedAt = now
	template.UpdatedAt = now

	// Store template
	m.templates[template.Name] = m.copyTemplate(template)

	m.logger.Info("Email template created", "name", template.Name)
	return m.copyTemplate(template), nil
}

// UpdateTemplate updates an existing template
func (m *EmailTemplateManager) UpdateTemplate(ctx context.Context, template *models.EmailTemplate) (*models.EmailTemplate, error) {
	if err := m.ValidateTemplate(ctx, template); err != nil {
		return nil, fmt.Errorf("template validation failed: %w", err)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if template exists
	existing, exists := m.templates[template.Name]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", template.Name)
	}

	// Preserve creation time
	template.CreatedAt = existing.CreatedAt
	template.UpdatedAt = time.Now()

	// Store updated template
	m.templates[template.Name] = m.copyTemplate(template)

	m.logger.Info("Email template updated", "name", template.Name)
	return m.copyTemplate(template), nil
}

// DeleteTemplate deletes a template
func (m *EmailTemplateManager) DeleteTemplate(ctx context.Context, name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if template exists
	if _, exists := m.templates[name]; !exists {
		return fmt.Errorf("template not found: %s", name)
	}

	// Don't allow deletion of default templates
	if m.isDefaultTemplate(name) {
		return fmt.Errorf("cannot delete default template: %s", name)
	}

	delete(m.templates, name)

	m.logger.Info("Email template deleted", "name", name)
	return nil
}

// ListTemplates lists templates with optional filtering
func (m *EmailTemplateManager) ListTemplates(ctx context.Context, filter *models.TemplateFilter) ([]*models.EmailTemplate, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var result []*models.EmailTemplate

	for _, template := range m.templates {
		if m.matchesFilter(template, filter) {
			result = append(result, m.copyTemplate(template))
		}
	}

	return result, nil
}

// ValidateTemplate validates a template
func (m *EmailTemplateManager) ValidateTemplate(ctx context.Context, template *models.EmailTemplate) error {
	if template == nil {
		return fmt.Errorf("template cannot be nil")
	}

	if template.Name == "" {
		return fmt.Errorf("template name cannot be empty")
	}

	if !isValidTemplateName(template.Name) {
		return fmt.Errorf("invalid template name: %s (must be alphanumeric with underscores)", template.Name)
	}

	if template.Subject == "" {
		return fmt.Errorf("template subject cannot be empty")
	}

	if template.Body == "" {
		return fmt.Errorf("template body cannot be empty")
	}

	// Validate that all declared variables are actually used in the template
	if err := m.validateTemplateVariables(template); err != nil {
		return err
	}

	// Validate template syntax
	if err := m.validateTemplateSyntax(template); err != nil {
		return err
	}

	return nil
}

// TestTemplate tests a template with sample data
func (m *EmailTemplateManager) TestTemplate(ctx context.Context, templateName string, testData map[string]interface{}) (*models.TemplateTestResult, error) {
	template, err := m.GetTemplate(ctx, templateName)
	if err != nil {
		return nil, err
	}

	result := &models.TemplateTestResult{
		TemplateName: templateName,
		TestData:     testData,
		Success:      true,
		TestedAt:     time.Now(),
	}

	// Process subject
	processedSubject, err := m.processTemplateString(template.Subject, testData)
	if err != nil {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("Subject processing failed: %v", err)
		return result, nil
	}
	result.ProcessedSubject = processedSubject

	// Process body
	processedBody, err := m.processTemplateString(template.Body, testData)
	if err != nil {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("Body processing failed: %v", err)
		return result, nil
	}
	result.ProcessedBody = processedBody

	// Check for unresolved variables
	unresolvedVars := m.findUnresolvedVariables(processedSubject + " " + processedBody)
	if len(unresolvedVars) > 0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Unresolved variables: %v", unresolvedVars))
	}

	return result, nil
}

// DuplicateTemplate creates a copy of an existing template
func (m *EmailTemplateManager) DuplicateTemplate(ctx context.Context, sourceName, newName string) (*models.EmailTemplate, error) {
	sourceTemplate, err := m.GetTemplate(ctx, sourceName)
	if err != nil {
		return nil, fmt.Errorf("source template not found: %w", err)
	}

	// Create new template based on source
	newTemplate := &models.EmailTemplate{
		Name:      newName,
		Subject:   sourceTemplate.Subject,
		Body:      sourceTemplate.Body,
		IsHTML:    sourceTemplate.IsHTML,
		Variables: make([]string, len(sourceTemplate.Variables)),
		IsActive:  sourceTemplate.IsActive,
	}

	// Copy variables slice
	copy(newTemplate.Variables, sourceTemplate.Variables)

	return m.CreateTemplate(ctx, newTemplate)
}

// GetTemplateVariables extracts variables from a template
func (m *EmailTemplateManager) GetTemplateVariables(ctx context.Context, templateName string) ([]string, error) {
	template, err := m.GetTemplate(ctx, templateName)
	if err != nil {
		return nil, err
	}

	// Extract variables from template content
	variables := m.extractVariablesFromContent(template.Subject + " " + template.Body)
	return variables, nil
}

// BackupTemplates creates a backup of all templates
func (m *EmailTemplateManager) BackupTemplates(ctx context.Context) ([]*models.EmailTemplate, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var backup []*models.EmailTemplate
	for _, template := range m.templates {
		backup = append(backup, m.copyTemplate(template))
	}

	m.logger.Info("Email templates backed up", "count", len(backup))
	return backup, nil
}

// RestoreTemplates restores templates from a backup
func (m *EmailTemplateManager) RestoreTemplates(ctx context.Context, templates []*models.EmailTemplate) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Validate all templates first
	for _, template := range templates {
		if err := m.ValidateTemplate(ctx, template); err != nil {
			return fmt.Errorf("invalid template %s: %w", template.Name, err)
		}
	}

	// Clear existing templates (except defaults)
	for name := range m.templates {
		if !m.isDefaultTemplate(name) {
			delete(m.templates, name)
		}
	}

	// Restore templates
	for _, template := range templates {
		m.templates[template.Name] = m.copyTemplate(template)
	}

	m.logger.Info("Email templates restored", "count", len(templates))
	return nil
}

// Private helper methods

func (m *EmailTemplateManager) loadDefaultTemplates() {
	defaults := []string{"overdue_reminder", "due_soon", "book_available", "fine_notice"}

	for _, name := range defaults {
		if defaultTemplate := GetDefaultTemplate(name); defaultTemplate != nil {
			// Convert to internal format
			template := &models.EmailTemplate{
				Name:      defaultTemplate.Name,
				Subject:   defaultTemplate.Subject,
				Body:      defaultTemplate.Body,
				IsHTML:    defaultTemplate.IsHTML,
				Variables: make([]string, len(defaultTemplate.Variables)),
				IsActive:  defaultTemplate.IsActive,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			copy(template.Variables, defaultTemplate.Variables)

			m.templates[name] = template
		}
	}

	m.logger.Info("Default email templates loaded", "count", len(m.templates))
}

func (m *EmailTemplateManager) copyTemplate(template *models.EmailTemplate) *models.EmailTemplate {
	if template == nil {
		return nil
	}

	templateCopy := &models.EmailTemplate{
		ID:        template.ID,
		Name:      template.Name,
		Subject:   template.Subject,
		Body:      template.Body,
		IsHTML:    template.IsHTML,
		Variables: make([]string, len(template.Variables)),
		IsActive:  template.IsActive,
		CreatedAt: template.CreatedAt,
		UpdatedAt: template.UpdatedAt,
	}

	// Deep copy variables slice
	if template.Variables != nil {
		templateCopy.Variables = make([]string, len(template.Variables))
		copy(templateCopy.Variables, template.Variables)
	}

	return templateCopy
}

func (m *EmailTemplateManager) isDefaultTemplate(name string) bool {
	defaults := []string{"overdue_reminder", "due_soon", "book_available", "fine_notice"}
	for _, defaultName := range defaults {
		if name == defaultName {
			return true
		}
	}
	return false
}

func (m *EmailTemplateManager) matchesFilter(template *models.EmailTemplate, filter *models.TemplateFilter) bool {
	if filter == nil {
		return true
	}

	if filter.IsActive != nil && template.IsActive != *filter.IsActive {
		return false
	}

	if filter.IsHTML != nil && template.IsHTML != *filter.IsHTML {
		return false
	}

	if filter.NamePattern != "" {
		if !strings.Contains(strings.ToLower(template.Name), strings.ToLower(filter.NamePattern)) {
			return false
		}
	}

	return true
}

func (m *EmailTemplateManager) validateTemplateVariables(template *models.EmailTemplate) error {
	content := template.Subject + " " + template.Body
	usedVariables := m.extractVariablesFromContent(content)

	// Check for declared but unused variables
	for _, declared := range template.Variables {
		found := false
		for _, used := range usedVariables {
			if declared == used {
				found = true
				break
			}
		}
		if !found {
			m.logger.Warn("Declared variable not used in template",
				"template", template.Name,
				"variable", declared)
		}
	}

	// Check for used but undeclared variables
	for _, used := range usedVariables {
		found := false
		for _, declared := range template.Variables {
			if used == declared {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("variable %s is used but not declared in template variables", used)
		}
	}

	return nil
}

func (m *EmailTemplateManager) validateTemplateSyntax(template *models.EmailTemplate) error {
	// Basic syntax validation - check for unmatched braces
	content := template.Subject + " " + template.Body

	openCount := strings.Count(content, "{{")
	closeCount := strings.Count(content, "}}")

	if openCount != closeCount {
		return fmt.Errorf("unmatched template braces in template %s", template.Name)
	}

	// Check for malformed variable syntax
	parts := strings.Split(content, "{{")
	for i := 1; i < len(parts); i++ {
		if !strings.Contains(parts[i], "}}") {
			return fmt.Errorf("malformed template variable syntax in template %s", template.Name)
		}
	}

	return nil
}

func (m *EmailTemplateManager) processTemplateString(template string, data map[string]interface{}) (string, error) {
	result := template

	if data != nil {
		for key, value := range data {
			placeholder := fmt.Sprintf("{{.%s}}", key)
			replacement := fmt.Sprintf("%v", value)
			result = strings.ReplaceAll(result, placeholder, replacement)
		}
	}

	return result, nil
}

func (m *EmailTemplateManager) extractVariablesFromContent(content string) []string {
	var variables []string

	parts := strings.Split(content, "{{.")
	for i := 1; i < len(parts); i++ {
		if endIndex := strings.Index(parts[i], "}}"); endIndex != -1 {
			variable := parts[i][:endIndex]
			// Only add if not already in list
			found := false
			for _, existing := range variables {
				if existing == variable {
					found = true
					break
				}
			}
			if !found {
				variables = append(variables, variable)
			}
		}
	}

	return variables
}

func (m *EmailTemplateManager) findUnresolvedVariables(content string) []string {
	var unresolved []string

	parts := strings.Split(content, "{{.")
	for i := 1; i < len(parts); i++ {
		if endIndex := strings.Index(parts[i], "}}"); endIndex != -1 {
			variable := parts[i][:endIndex]
			// Only add if not already in list
			found := false
			for _, existing := range unresolved {
				if existing == variable {
					found = true
					break
				}
			}
			if !found {
				unresolved = append(unresolved, variable)
			}
		}
	}

	return unresolved
}

func isValidTemplateName(name string) bool {
	if name == "" {
		return false
	}

	for _, char := range name {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_') {
			return false
		}
	}

	return true
}
