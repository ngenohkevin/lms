package handlers

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Success bool        `json:"success"`
	Error   ErrorDetail `json:"error"`
}

// ErrorDetail represents detailed error information
type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// SuccessResponse represents a standard success response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// ListResponse represents a paginated list response
type ListResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Meta    interface{} `json:"meta,omitempty"`
}