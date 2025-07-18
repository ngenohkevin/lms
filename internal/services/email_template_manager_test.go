package services

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/ngenohkevin/lms/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailTemplateManager(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	manager := NewEmailTemplateManager(logger)
	ctx := context.Background()

	t.Run("Load default templates", func(t *testing.T) {
		// Check that default templates are loaded
		templates, err := manager.ListTemplates(ctx, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(templates), 4) // At least 4 default templates

		// Verify specific default templates exist
		defaultNames := []string{"overdue_reminder", "due_soon", "book_available", "fine_notice"}
		for _, name := range defaultNames {
			template, err := manager.GetTemplate(ctx, name)
			assert.NoError(t, err)
			assert.NotNil(t, template)
			assert.Equal(t, name, template.Name)
		}
	})

	t.Run("Create new template", func(t *testing.T) {
		template := &models.EmailTemplate{
			Name:      "test_template",
			Subject:   "Test Subject - {{.Name}}",
			Body:      "Hello {{.Name}}, this is a test message about {{.Topic}}.",
			IsHTML:    false,
			Variables: []string{"Name", "Topic"},
			IsActive:  true,
		}

		created, err := manager.CreateTemplate(ctx, template)
		require.NoError(t, err)
		assert.Equal(t, template.Name, created.Name)
		assert.Equal(t, template.Subject, created.Subject)
		assert.Equal(t, template.Body, created.Body)
		assert.False(t, created.CreatedAt.IsZero())
		assert.False(t, created.UpdatedAt.IsZero())

		// Verify template can be retrieved
		retrieved, err := manager.GetTemplate(ctx, "test_template")
		require.NoError(t, err)
		assert.Equal(t, created.Name, retrieved.Name)
	})

	t.Run("Update template", func(t *testing.T) {
		// First create a template
		template := &models.EmailTemplate{
			Name:      "update_test",
			Subject:   "Original Subject",
			Body:      "Original body with {{.Variable}}",
			IsHTML:    false,
			Variables: []string{"Variable"},
			IsActive:  true,
		}

		created, err := manager.CreateTemplate(ctx, template)
		require.NoError(t, err)

		// Update the template
		created.Subject = "Updated Subject"
		created.Body = "Updated body with {{.Variable}} and {{.NewVar}}"
		created.Variables = []string{"Variable", "NewVar"}

		updated, err := manager.UpdateTemplate(ctx, created)
		require.NoError(t, err)
		assert.Equal(t, "Updated Subject", updated.Subject)
		assert.Contains(t, updated.Body, "Updated body")
		assert.Contains(t, updated.Variables, "NewVar")
		assert.True(t, updated.UpdatedAt.After(updated.CreatedAt))
	})

	t.Run("Delete template", func(t *testing.T) {
		// Create a template to delete
		template := &models.EmailTemplate{
			Name:      "delete_test",
			Subject:   "Subject",
			Body:      "Body",
			IsHTML:    false,
			Variables: []string{},
			IsActive:  true,
		}

		_, err := manager.CreateTemplate(ctx, template)
		require.NoError(t, err)

		// Delete the template
		err = manager.DeleteTemplate(ctx, "delete_test")
		assert.NoError(t, err)

		// Verify template is deleted
		_, err = manager.GetTemplate(ctx, "delete_test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "template not found")
	})

	t.Run("Cannot delete default template", func(t *testing.T) {
		err := manager.DeleteTemplate(ctx, "overdue_reminder")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete default template")
	})

	t.Run("Duplicate template", func(t *testing.T) {
		// Duplicate a default template
		duplicated, err := manager.DuplicateTemplate(ctx, "due_soon", "due_soon_copy")
		require.NoError(t, err)
		assert.Equal(t, "due_soon_copy", duplicated.Name)

		// Verify original template content was copied
		original, err := manager.GetTemplate(ctx, "due_soon")
		require.NoError(t, err)
		assert.Equal(t, original.Subject, duplicated.Subject)
		assert.Equal(t, original.Body, duplicated.Body)
		assert.Equal(t, original.Variables, duplicated.Variables)
	})
}

func TestEmailTemplateManagerValidation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	manager := NewEmailTemplateManager(logger)
	ctx := context.Background()

	tests := []struct {
		name        string
		template    *models.EmailTemplate
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid template",
			template: &models.EmailTemplate{
				Name:      "valid_template",
				Subject:   "Subject with {{.Variable}}",
				Body:      "Body with {{.Variable}}",
				IsHTML:    false,
				Variables: []string{"Variable"},
				IsActive:  true,
			},
			expectError: false,
		},
		{
			name:        "Nil template",
			template:    nil,
			expectError: true,
			errorMsg:    "template cannot be nil",
		},
		{
			name: "Empty name",
			template: &models.EmailTemplate{
				Name:      "",
				Subject:   "Subject",
				Body:      "Body",
				IsHTML:    false,
				Variables: []string{},
				IsActive:  true,
			},
			expectError: true,
			errorMsg:    "template name cannot be empty",
		},
		{
			name: "Invalid name characters",
			template: &models.EmailTemplate{
				Name:      "invalid-name!",
				Subject:   "Subject",
				Body:      "Body",
				IsHTML:    false,
				Variables: []string{},
				IsActive:  true,
			},
			expectError: true,
			errorMsg:    "invalid template name",
		},
		{
			name: "Empty subject",
			template: &models.EmailTemplate{
				Name:      "test_template",
				Subject:   "",
				Body:      "Body",
				IsHTML:    false,
				Variables: []string{},
				IsActive:  true,
			},
			expectError: true,
			errorMsg:    "template subject cannot be empty",
		},
		{
			name: "Empty body",
			template: &models.EmailTemplate{
				Name:      "test_template",
				Subject:   "Subject",
				Body:      "",
				IsHTML:    false,
				Variables: []string{},
				IsActive:  true,
			},
			expectError: true,
			errorMsg:    "template body cannot be empty",
		},
		{
			name: "Used but undeclared variable",
			template: &models.EmailTemplate{
				Name:      "undeclared_var",
				Subject:   "Subject with {{.UndeclaredVar}}",
				Body:      "Body",
				IsHTML:    false,
				Variables: []string{},
				IsActive:  true,
			},
			expectError: true,
			errorMsg:    "variable UndeclaredVar is used but not declared",
		},
		{
			name: "Unmatched braces",
			template: &models.EmailTemplate{
				Name:      "unmatched_braces",
				Subject:   "Subject with {{.Variable",
				Body:      "Body",
				IsHTML:    false,
				Variables: []string{"Variable"},
				IsActive:  true,
			},
			expectError: true,
			errorMsg:    "unmatched template braces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.ValidateTemplate(ctx, tt.template)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEmailTemplateFiltering(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	manager := NewEmailTemplateManager(logger)
	ctx := context.Background()

	// Create test templates
	templates := []*models.EmailTemplate{
		{
			Name:      "active_text",
			Subject:   "Subject",
			Body:      "Body",
			IsHTML:    false,
			Variables: []string{},
			IsActive:  true,
		},
		{
			Name:      "inactive_text",
			Subject:   "Subject",
			Body:      "Body",
			IsHTML:    false,
			Variables: []string{},
			IsActive:  false,
		},
		{
			Name:      "active_html",
			Subject:   "Subject",
			Body:      "<p>Body</p>",
			IsHTML:    true,
			Variables: []string{},
			IsActive:  true,
		},
	}

	for _, template := range templates {
		_, err := manager.CreateTemplate(ctx, template)
		require.NoError(t, err)
	}

	tests := []struct {
		name          string
		filter        *models.TemplateFilter
		expectedNames []string
		minResults    int
	}{
		{
			name:       "No filter",
			filter:     nil,
			minResults: 3, // At least our test templates plus defaults
		},
		{
			name: "Active templates only",
			filter: &models.TemplateFilter{
				IsActive: &[]bool{true}[0],
			},
			expectedNames: []string{"active_text", "active_html"},
		},
		{
			name: "Inactive templates only",
			filter: &models.TemplateFilter{
				IsActive: &[]bool{false}[0],
			},
			expectedNames: []string{"inactive_text"},
		},
		{
			name: "HTML templates only",
			filter: &models.TemplateFilter{
				IsHTML: &[]bool{true}[0],
			},
			expectedNames: []string{"active_html"},
		},
		{
			name: "Text templates only",
			filter: &models.TemplateFilter{
				IsHTML: &[]bool{false}[0],
			},
			minResults: 2, // Our text templates plus defaults
		},
		{
			name: "Name pattern filter",
			filter: &models.TemplateFilter{
				NamePattern: "active",
			},
			expectedNames: []string{"active_text", "active_html"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := manager.ListTemplates(ctx, tt.filter)
			require.NoError(t, err)

			if tt.expectedNames != nil {
				// Check for specific template names
				resultNames := make([]string, len(results))
				for i, template := range results {
					resultNames[i] = template.Name
				}

				for _, expectedName := range tt.expectedNames {
					assert.Contains(t, resultNames, expectedName)
				}
			}

			if tt.minResults > 0 {
				assert.GreaterOrEqual(t, len(results), tt.minResults)
			}
		})
	}
}

func TestEmailTemplateTest(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	manager := NewEmailTemplateManager(logger)
	ctx := context.Background()

	// Create a test template
	template := &models.EmailTemplate{
		Name:      "test_template",
		Subject:   "Welcome {{.Name}} to {{.Library}}",
		Body:      "Dear {{.Name}},\n\nWelcome to {{.Library}}! Your account has been created.\n\nBest regards,\nThe {{.Library}} Team",
		IsHTML:    false,
		Variables: []string{"Name", "Library"},
		IsActive:  true,
	}

	_, err := manager.CreateTemplate(ctx, template)
	require.NoError(t, err)

	tests := []struct {
		name         string
		templateName string
		testData     map[string]interface{}
		expectError  bool
		checkContent bool
	}{
		{
			name:         "Valid test data",
			templateName: "test_template",
			testData: map[string]interface{}{
				"Name":    "John Doe",
				"Library": "Central Library",
			},
			expectError:  false,
			checkContent: true,
		},
		{
			name:         "Missing template",
			templateName: "non_existent",
			testData:     map[string]interface{}{},
			expectError:  true,
		},
		{
			name:         "Partial data",
			templateName: "test_template",
			testData: map[string]interface{}{
				"Name": "Jane Smith",
				// Library missing
			},
			expectError:  false,
			checkContent: true,
		},
		{
			name:         "Empty data",
			templateName: "test_template",
			testData:     map[string]interface{}{},
			expectError:  false,
			checkContent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.TestTemplate(ctx, tt.templateName, tt.testData)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.templateName, result.TemplateName)
			assert.False(t, result.TestedAt.IsZero())

			if tt.checkContent {
				assert.NotEmpty(t, result.ProcessedSubject)
				assert.NotEmpty(t, result.ProcessedBody)

				// Check if variables were replaced
				if tt.testData != nil {
					for _, value := range tt.testData {
						valueStr := value.(string)
						assert.Contains(t, result.ProcessedSubject+result.ProcessedBody, valueStr)
					}
				}
			}
		})
	}
}

func TestEmailTemplateVariableExtraction(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	manager := NewEmailTemplateManager(logger)
	ctx := context.Background()

	// Create a template with variables
	template := &models.EmailTemplate{
		Name:      "variable_test",
		Subject:   "{{.Title}} - {{.Subject}}",
		Body:      "Hello {{.Name}}, your {{.ItemType}} {{.ItemName}} is {{.Status}}. Due date: {{.DueDate}}",
		IsHTML:    false,
		Variables: []string{"Title", "Subject", "Name", "ItemType", "ItemName", "Status", "DueDate"},
		IsActive:  true,
	}

	_, err := manager.CreateTemplate(ctx, template)
	require.NoError(t, err)

	variables, err := manager.GetTemplateVariables(ctx, "variable_test")
	require.NoError(t, err)

	expectedVariables := []string{"Title", "Subject", "Name", "ItemType", "ItemName", "Status", "DueDate"}

	// Check that all expected variables are found
	for _, expected := range expectedVariables {
		assert.Contains(t, variables, expected, "Should contain variable %s", expected)
	}

	// Check that no extra variables are found
	assert.Equal(t, len(expectedVariables), len(variables), "Should have exactly %d variables", len(expectedVariables))
}

func TestEmailTemplateBackupRestore(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	manager := NewEmailTemplateManager(logger)
	ctx := context.Background()

	// Create some custom templates
	customTemplates := []*models.EmailTemplate{
		{
			Name:      "custom1",
			Subject:   "Custom 1",
			Body:      "Body 1",
			IsHTML:    false,
			Variables: []string{},
			IsActive:  true,
		},
		{
			Name:      "custom2",
			Subject:   "Custom 2",
			Body:      "Body 2",
			IsHTML:    true,
			Variables: []string{},
			IsActive:  false,
		},
	}

	for _, template := range customTemplates {
		_, err := manager.CreateTemplate(ctx, template)
		require.NoError(t, err)
	}

	// Create backup
	backup, err := manager.BackupTemplates(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(backup), 2) // At least our custom templates

	// Verify custom templates are in backup
	customNames := []string{"custom1", "custom2"}
	backupNames := make([]string, len(backup))
	for i, template := range backup {
		backupNames[i] = template.Name
	}

	for _, name := range customNames {
		assert.Contains(t, backupNames, name)
	}

	// Delete a custom template
	err = manager.DeleteTemplate(ctx, "custom1")
	require.NoError(t, err)

	// Restore from backup
	err = manager.RestoreTemplates(ctx, backup)
	require.NoError(t, err)

	// Verify template was restored
	restored, err := manager.GetTemplate(ctx, "custom1")
	require.NoError(t, err)
	assert.Equal(t, "custom1", restored.Name)
}
