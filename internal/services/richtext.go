package services

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// RichTextServiceInterface defines the interface for rich text operations
type RichTextServiceInterface interface {
	SanitizeHTML(content string) string
	ValidateHTML(content string) error
	StripHTML(content string) string
	ExtractPlainText(content string) string
	IsValidRichText(content string) bool
}

// RichTextService handles rich text processing and sanitization
type RichTextService struct {
	// Define allowed HTML tags and attributes for security
	allowedTags map[string][]string
	maxLength   int
}

// NewRichTextService creates a new rich text service
func NewRichTextService() *RichTextService {
	return &RichTextService{
		allowedTags: map[string][]string{
			"p":          {},
			"br":         {},
			"strong":     {},
			"em":         {},
			"b":          {},
			"i":          {},
			"u":          {},
			"ul":         {},
			"ol":         {},
			"li":         {},
			"h1":         {},
			"h2":         {},
			"h3":         {},
			"h4":         {},
			"h5":         {},
			"h6":         {},
			"div":        {"class"},
			"span":       {"class"},
			"a":          {"href", "title", "target"},
			"img":        {"src", "alt", "title", "width", "height"},
			"table":      {},
			"tr":         {},
			"td":         {},
			"th":         {},
			"thead":      {},
			"tbody":      {},
			"blockquote": {},
		},
		maxLength: 10000, // Maximum length for rich text content
	}
}

// SanitizeHTML sanitizes HTML content by removing dangerous elements
func (s *RichTextService) SanitizeHTML(content string) string {
	if content == "" {
		return ""
	}

	// Remove script tags and their content (simplified approach)
	scriptRegex := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	content = scriptRegex.ReplaceAllString(content, "")

	// Remove style tags and their content (simplified approach)
	styleRegex := regexp.MustCompile(`(?i)<style[^>]*>.*?</style>`)
	content = styleRegex.ReplaceAllString(content, "")

	// Remove dangerous attributes like onclick, onload, etc.
	onEventRegex := regexp.MustCompile(`(?i)\s+on\w+\s*=\s*["\'][^"\']*["\']`)
	content = onEventRegex.ReplaceAllString(content, "")

	// Remove javascript: URLs
	jsRegex := regexp.MustCompile(`(?i)javascript\s*:`)
	content = jsRegex.ReplaceAllString(content, "")

	// Remove vbscript: URLs
	vbsRegex := regexp.MustCompile(`(?i)vbscript\s*:`)
	content = vbsRegex.ReplaceAllString(content, "")

	// Remove data: URLs for security (simple approach)
	dataRegex := regexp.MustCompile(`(?i)data:[^;\s]*;`)
	content = dataRegex.ReplaceAllString(content, "")

	// Sanitize attributes in allowed tags
	content = s.sanitizeAttributes(content)

	// Remove disallowed tags
	content = s.removeDisallowedTags(content)

	// Limit length
	if len(content) > s.maxLength {
		content = content[:s.maxLength]
		// Try to end at a complete tag
		lastTagIndex := strings.LastIndex(content, ">")
		if lastTagIndex > 0 && lastTagIndex < len(content)-100 {
			content = content[:lastTagIndex+1]
		}
	}

	return strings.TrimSpace(content)
}

// ValidateHTML validates HTML content structure and security
func (s *RichTextService) ValidateHTML(content string) error {
	if len(content) > s.maxLength {
		return fmt.Errorf("content exceeds maximum length of %d characters", s.maxLength)
	}

	// Check for suspicious patterns
	suspiciousPatterns := []string{
		`<script`,
		`javascript:`,
		`vbscript:`,
		`onload=`,
		`onclick=`,
		`onerror=`,
		`onmouseover=`,
		`<iframe`,
		`<object`,
		`<embed`,
		`<link`,
		`<meta`,
		`<base`,
		`<form`,
		`<input`,
		`<textarea`,
		`<select`,
		`<button`,
	}

	lowerContent := strings.ToLower(content)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerContent, pattern) {
			return fmt.Errorf("content contains potentially dangerous HTML: %s", pattern)
		}
	}

	// Check for balanced tags (basic validation)
	if err := s.validateHTMLStructure(content); err != nil {
		return fmt.Errorf("invalid HTML structure: %w", err)
	}

	return nil
}

// StripHTML removes all HTML tags and returns plain text
func (s *RichTextService) StripHTML(content string) string {
	if content == "" {
		return ""
	}

	// Remove all HTML tags
	htmlTagRegex := regexp.MustCompile(`<[^>]*>`)
	plainText := htmlTagRegex.ReplaceAllString(content, "")

	// Unescape HTML entities
	plainText = html.UnescapeString(plainText)

	// Clean up whitespace
	spaceRegex := regexp.MustCompile(`\s+`)
	plainText = spaceRegex.ReplaceAllString(plainText, " ")

	return strings.TrimSpace(plainText)
}

// ExtractPlainText extracts plain text while preserving some formatting
func (s *RichTextService) ExtractPlainText(content string) string {
	if content == "" {
		return ""
	}

	// Replace block elements with newlines
	blockElements := []string{"p", "div", "br", "h1", "h2", "h3", "h4", "h5", "h6", "li"}
	for _, elem := range blockElements {
		openTag := fmt.Sprintf("<%s[^>]*>", elem)
		closeTag := fmt.Sprintf("</%s>", elem)

		openRegex := regexp.MustCompile(fmt.Sprintf(`(?i)%s`, openTag))
		closeRegex := regexp.MustCompile(fmt.Sprintf(`(?i)%s`, closeTag))

		content = openRegex.ReplaceAllString(content, "")
		content = closeRegex.ReplaceAllString(content, "\n")
	}

	// Handle self-closing br tags
	brRegex := regexp.MustCompile(`(?i)<br\s*/?>`)
	content = brRegex.ReplaceAllString(content, "\n")

	// Remove remaining HTML tags
	return s.StripHTML(content)
}

// IsValidRichText checks if the content is valid rich text
func (s *RichTextService) IsValidRichText(content string) bool {
	if content == "" {
		return true
	}

	// Check if content contains HTML tags
	htmlTagRegex := regexp.MustCompile(`<[^>]*>`)
	if !htmlTagRegex.MatchString(content) {
		// Plain text is valid
		return true
	}

	// Validate HTML content
	return s.ValidateHTML(content) == nil
}

// sanitizeAttributes removes dangerous attributes from HTML tags
func (s *RichTextService) sanitizeAttributes(content string) string {
	// This is a simple implementation - in production, you might want to use a proper HTML parser
	tagRegex := regexp.MustCompile(`<([^>]+)>`)

	return tagRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Extract tag name
		tagContent := match[1 : len(match)-1]
		parts := strings.Fields(tagContent)
		if len(parts) == 0 {
			return match
		}

		tagName := strings.ToLower(parts[0])
		if tagName[0] == '/' {
			// Closing tag, no attributes to sanitize
			return match
		}

		// Check if tag is allowed
		allowedAttrs, tagAllowed := s.allowedTags[tagName]
		if !tagAllowed {
			return ""
		}

		// If no attributes are allowed for this tag, return just the tag
		if len(allowedAttrs) == 0 {
			return fmt.Sprintf("<%s>", tagName)
		}

		// Sanitize attributes (simplified approach)
		result := tagName
		for i := 1; i < len(parts); i++ {
			attr := parts[i]
			if strings.Contains(attr, "=") {
				attrName := strings.Split(attr, "=")[0]
				attrName = strings.ToLower(attrName)

				// Check if attribute is allowed
				for _, allowed := range allowedAttrs {
					if allowed == attrName {
						result += " " + attr
						break
					}
				}
			}
		}

		return fmt.Sprintf("<%s>", result)
	})
}

// removeDisallowedTags removes HTML tags that are not in the allowed list
func (s *RichTextService) removeDisallowedTags(content string) string {
	// Remove complete tags (opening and closing) that are not allowed
	tagRegex := regexp.MustCompile(`</?([a-zA-Z][a-zA-Z0-9]*)\b[^>]*>`)

	return tagRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Extract tag name
		tagNameRegex := regexp.MustCompile(`</?([a-zA-Z][a-zA-Z0-9]*)\b`)
		tagNameMatch := tagNameRegex.FindStringSubmatch(match)
		if len(tagNameMatch) < 2 {
			return ""
		}

		tagName := strings.ToLower(tagNameMatch[1])

		// Check if tag is allowed
		if _, allowed := s.allowedTags[tagName]; allowed {
			return match
		}

		return ""
	})
}

// validateHTMLStructure performs basic HTML structure validation
func (s *RichTextService) validateHTMLStructure(content string) error {
	// Simple stack-based validation for balanced tags
	stack := []string{}
	tagRegex := regexp.MustCompile(`<(/?)([a-zA-Z][a-zA-Z0-9]*)\b[^>]*(/?)>`)

	matches := tagRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		isClosing := match[1] == "/"
		tagName := strings.ToLower(match[2])
		isSelfClosing := match[3] == "/" || s.isSelfClosingTag(tagName)

		if isClosing {
			// Closing tag
			if len(stack) == 0 {
				return fmt.Errorf("unexpected closing tag: %s", tagName)
			}

			lastTag := stack[len(stack)-1]
			if lastTag != tagName {
				return fmt.Errorf("mismatched closing tag: expected %s, got %s", lastTag, tagName)
			}

			stack = stack[:len(stack)-1]
		} else if !isSelfClosing {
			// Opening tag (not self-closing)
			stack = append(stack, tagName)
		}
		// Self-closing tags don't need to be pushed to stack
	}

	if len(stack) > 0 {
		return fmt.Errorf("unclosed tags: %v", stack)
	}

	return nil
}

// isSelfClosingTag checks if a tag is self-closing
func (s *RichTextService) isSelfClosingTag(tagName string) bool {
	selfClosingTags := map[string]bool{
		"br":     true,
		"hr":     true,
		"img":    true,
		"input":  true,
		"meta":   true,
		"link":   true,
		"area":   true,
		"base":   true,
		"col":    true,
		"embed":  true,
		"source": true,
		"track":  true,
		"wbr":    true,
	}

	return selfClosingTags[tagName]
}
