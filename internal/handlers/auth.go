package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/middleware"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
)

type AuthHandler struct {
	authService  *services.AuthService
	userService  services.UserServiceInterface
	emailService services.EmailServiceInterface
	auditLogger  *middleware.AuditLogger
}

func NewAuthHandler(authService *services.AuthService, userService services.UserServiceInterface, emailService services.EmailServiceInterface, auditLogger *middleware.AuditLogger) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		userService:  userService,
		emailService: emailService,
		auditLogger:  auditLogger,
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest

	// Always return same error for invalid credentials to prevent timing attacks
	invalidCredentialsResponse := gin.H{
		"success": false,
		"error": gin.H{
			"code":    "INVALID_CREDENTIALS",
			"message": "Invalid username or password",
		},
	}

	// Parse JSON but handle validation errors securely
	if err := c.ShouldBindJSON(&req); err != nil {
		// Perform dummy password verification to maintain consistent timing
		dummyHash := "$argon2id$v=19$m=65536,t=3,p=2$wST4GhL1dnC5T2ZYRsn+Rg$FLCDqu8e9OL9XtJDkMVWRxMXUjr5b4KVG+VZYsEDr8g"
		_, _ = h.authService.VerifyPassword(dummyHash, "dummy")
		// Return unauthorized instead of bad request to prevent information leakage
		c.JSON(http.StatusUnauthorized, invalidCredentialsResponse)
		return
	}

	// Handle empty username/password consistently
	if req.Username == "" || req.Password == "" {
		// Perform dummy password verification to maintain consistent timing
		// Use a more realistic dummy hash and the actual password provided for timing consistency
		dummyHash := "$argon2id$v=19$m=65536,t=3,p=2$wST4GhL1dnC5T2ZYRsn+Rg$FLCDqu8e9OL9XtJDkMVWRxMXUjr5b4KVG+VZYsEDr8g"
		if req.Password != "" {
			_, _ = h.authService.VerifyPassword(dummyHash, req.Password)
		} else {
			_, _ = h.authService.VerifyPassword(dummyHash, "dummy")
		}
		c.JSON(http.StatusUnauthorized, invalidCredentialsResponse)
		return
	}

	// Try to authenticate as librarian first
	user, err := h.userService.GetUserByUsername(req.Username)
	if err == nil && user != nil {
		// Verify password
		isValid, err := h.authService.VerifyPassword(user.PasswordHash, req.Password)
		if err != nil {
			middleware.AuditAuth(c, h.auditLogger, "LOGIN_FAILED", nil, string(user.Role), map[string]interface{}{"username": req.Username, "reason": "password_verification_error"})
			c.JSON(http.StatusUnauthorized, invalidCredentialsResponse)
			return
		}

		if !isValid {
			userID := int32(user.ID)
			middleware.AuditAuth(c, h.auditLogger, "LOGIN_FAILED", &userID, string(user.Role), map[string]interface{}{"username": req.Username, "reason": "invalid_password"})
			c.JSON(http.StatusUnauthorized, invalidCredentialsResponse)
			return
		}

		if !user.IsActive {
			userID := int32(user.ID)
			middleware.AuditAuth(c, h.auditLogger, "LOGIN_FAILED", &userID, string(user.Role), map[string]interface{}{"username": req.Username, "reason": "account_inactive"})
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ACCOUNT_INACTIVE",
					"message": "Account is inactive",
				},
			})
			return
		}

		// Generate tokens for librarian
		accessToken, refreshToken, err := h.authService.GenerateTokens(user, string(user.Role))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "TOKEN_GENERATION_ERROR",
					"message": "Error generating tokens",
				},
			})
			return
		}

		// Update last login (non-critical - don't fail login if this fails)
		_ = h.userService.UpdateLastLogin(user.ID)

		userID := int32(user.ID)
		middleware.AuditAuth(c, h.auditLogger, "LOGIN", &userID, string(user.Role), map[string]interface{}{"username": req.Username, "role": user.Role})

		response := models.LoginResponse{
			User:         user,
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    3600, // 1 hour in seconds
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    response,
			"message": "Login successful",
		})
		return
	}

	// Try to authenticate as student
	student, err := h.userService.GetStudentByStudentID(req.Username)
	if err != nil {
		// Perform dummy password verification to maintain consistent timing
		dummyHash := "$argon2id$v=19$m=65536,t=3,p=2$wST4GhL1dnC5T2ZYRsn+Rg$FLCDqu8e9OL9XtJDkMVWRxMXUjr5b4KVG+VZYsEDr8g"
		_, _ = h.authService.VerifyPassword(dummyHash, req.Password)
		middleware.AuditAuth(c, h.auditLogger, "LOGIN_FAILED", nil, "unknown", map[string]interface{}{"username": req.Username, "reason": "user_not_found"})
		c.JSON(http.StatusUnauthorized, invalidCredentialsResponse)
		return
	}

	// For students, if no password is set, use student ID as default password
	studentID := int32(student.ID)
	if student.PasswordHash == nil {
		if req.Password != student.StudentID {
			middleware.AuditAuth(c, h.auditLogger, "LOGIN_FAILED", &studentID, "student", map[string]interface{}{"student_id": student.StudentID, "reason": "invalid_password"})
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_CREDENTIALS",
					"message": "Invalid username or password",
				},
			})
			return
		}
	} else {
		// Verify password
		isValid, err := h.authService.VerifyPassword(*student.PasswordHash, req.Password)
		if err != nil {
			middleware.AuditAuth(c, h.auditLogger, "LOGIN_FAILED", &studentID, "student", map[string]interface{}{"student_id": student.StudentID, "reason": "password_verification_error"})
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_CREDENTIALS",
					"message": "Invalid username or password",
				},
			})
			return
		}

		if !isValid {
			middleware.AuditAuth(c, h.auditLogger, "LOGIN_FAILED", &studentID, "student", map[string]interface{}{"student_id": student.StudentID, "reason": "invalid_password"})
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_CREDENTIALS",
					"message": "Invalid username or password",
				},
			})
			return
		}
	}

	if !student.IsActive {
		middleware.AuditAuth(c, h.auditLogger, "LOGIN_FAILED", &studentID, "student", map[string]interface{}{"student_id": student.StudentID, "reason": "account_inactive"})
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ACCOUNT_INACTIVE",
				"message": "Account is inactive",
			},
		})
		return
	}

	// Generate tokens for student
	accessToken, refreshToken, err := h.authService.GenerateStudentTokens(student)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "TOKEN_GENERATION_ERROR",
				"message": "Error generating tokens",
			},
		})
		return
	}

	middleware.AuditAuth(c, h.auditLogger, "LOGIN", &studentID, "student", map[string]interface{}{"student_id": student.StudentID})

	response := models.LoginResponse{
		Student:      student,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600, // 1 hour in seconds
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
		"message": "Login successful",
	})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req models.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid request data",
				"details": err.Error(),
			},
		})
		return
	}

	// Validate refresh token and generate new tokens
	newAccessToken, newRefreshToken, err := h.authService.RefreshTokens(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REFRESH_TOKEN",
				"message": "Invalid or expired refresh token",
			},
		})
		return
	}

	response := gin.H{
		"access_token":  newAccessToken,
		"refresh_token": newRefreshToken,
		"token_type":    "Bearer",
		"expires_in":    3600,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
		"message": "Tokens refreshed successfully",
	})
}

// LogoutRequest represents the logout request body
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Logout(c *gin.Context) {
	// Get access token from Authorization header
	authHeader := c.GetHeader("Authorization")
	var accessToken string
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			accessToken = parts[1]
		}
	}

	// Extract user info from token before blacklisting (for audit)
	var logoutUserID *int32
	logoutUserType := "unknown"
	if accessToken != "" {
		if claims, err := h.authService.ValidateToken(accessToken); err == nil {
			uid := int32(claims.UserID)
			logoutUserID = &uid
			logoutUserType = claims.UserType
		}
	}

	// Get refresh token from request body
	var req LogoutRequest
	// Bind JSON but don't fail if body is empty
	_ = c.ShouldBindJSON(&req)

	// Blacklist both tokens to ensure complete logout
	if accessToken != "" {
		if err := h.authService.BlacklistToken(accessToken); err != nil {
			// Log but don't fail - user is logged out client-side regardless
			slog.Warn("Failed to blacklist access token", "error", err)
		}
	}

	if req.RefreshToken != "" {
		if err := h.authService.BlacklistRefreshToken(req.RefreshToken); err != nil {
			// Log but don't fail - user is logged out client-side regardless
			slog.Warn("Failed to blacklist refresh token", "error", err)
		}
	}

	middleware.AuditAuth(c, h.auditLogger, "LOGOUT", logoutUserID, logoutUserType, nil)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logout successful",
	})
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userType := middleware.GetUserType(c)

	if userType == "student" {
		student, err := h.userService.GetStudentByID(userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "USER_NOT_FOUND",
					"message": "Student not found",
				},
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    student,
		})
		return
	}

	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "USER_NOT_FOUND",
				"message": "User not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
	})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid request data",
				"details": err.Error(),
			},
		})
		return
	}

	userID := middleware.GetUserID(c)
	userType := middleware.GetUserType(c)

	if userType == "student" {
		student, err := h.userService.GetStudentByID(userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "USER_NOT_FOUND",
					"message": "Student not found",
				},
			})
			return
		}

		// Verify current password
		if student.PasswordHash != nil {
			isValid, err := h.authService.VerifyPassword(*student.PasswordHash, req.CurrentPassword)
			if err != nil || !isValid {
				c.JSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "INVALID_CURRENT_PASSWORD",
						"message": "Current password is incorrect",
					},
				})
				return
			}
		} else {
			// If no password is set, verify against student ID
			if req.CurrentPassword != student.StudentID {
				c.JSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "INVALID_CURRENT_PASSWORD",
						"message": "Current password is incorrect",
					},
				})
				return
			}
		}

		// Hash new password
		hashedPassword, err := h.authService.HashPassword(req.NewPassword)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_PASSWORD",
					"message": "Password must be at least 8 characters long",
				},
			})
			return
		}

		// Update password
		err = h.userService.UpdateStudentPassword(userID, hashedPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UPDATE_ERROR",
					"message": "Error updating password",
				},
			})
			return
		}

		middleware.Audit(c, "students", int32(userID), "PASSWORD_CHANGE", nil, map[string]interface{}{"student_id": student.StudentID})
	} else {
		user, err := h.userService.GetUserByID(userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "USER_NOT_FOUND",
					"message": "User not found",
				},
			})
			return
		}

		// Verify current password
		isValid, err := h.authService.VerifyPassword(user.PasswordHash, req.CurrentPassword)
		if err != nil || !isValid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_CURRENT_PASSWORD",
					"message": "Current password is incorrect",
				},
			})
			return
		}

		// Hash new password
		hashedPassword, err := h.authService.HashPassword(req.NewPassword)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_PASSWORD",
					"message": "Password must be at least 8 characters long",
				},
			})
			return
		}

		// Update password
		err = h.userService.UpdatePassword(userID, hashedPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UPDATE_ERROR",
					"message": "Error updating password",
				},
			})
			return
		}

		middleware.Audit(c, "users", int32(userID), "PASSWORD_CHANGE", nil, map[string]interface{}{"username": user.Username})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Password updated successfully",
	})
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req models.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid request data",
				"details": err.Error(),
			},
		})
		return
	}

	// Check if user exists
	user, err := h.userService.GetUserByEmail(req.Email)
	if err != nil {
		// Don't reveal whether user exists or not for security
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "If an account with this email exists, a password reset link has been sent",
		})
		return
	}

	// Generate password reset token
	token, err := h.authService.GeneratePasswordResetToken(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Error generating reset token",
			},
		})
		return
	}

	// Send password reset email
	if err := h.sendPasswordResetEmail(c.Request.Context(), user, token); err != nil {
		slog.Error("Failed to send password reset email",
			"email", req.Email,
			"error", err)
	}

	userID := int32(user.ID)
	middleware.AuditAuth(c, h.auditLogger, "PASSWORD_RESET", &userID, "librarian", map[string]interface{}{"email": req.Email, "stage": "requested"})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "If an account with this email exists, a password reset link has been sent",
	})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req models.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid request data",
				"details": err.Error(),
			},
		})
		return
	}

	// Validate reset token
	email, err := h.authService.ValidatePasswordResetToken(req.Token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_RESET_TOKEN",
				"message": "Invalid or expired reset token",
			},
		})
		return
	}

	// Get user by email
	user, err := h.userService.GetUserByEmail(email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "USER_NOT_FOUND",
				"message": "User not found",
			},
		})
		return
	}

	// Hash new password
	hashedPassword, err := h.authService.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_PASSWORD",
				"message": "Password must be at least 8 characters long",
			},
		})
		return
	}

	// Update password
	err = h.userService.UpdatePassword(user.ID, hashedPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "UPDATE_ERROR",
				"message": "Error updating password",
			},
		})
		return
	}

	// Invalidate the reset token (non-critical - password already reset successfully)
	_ = h.authService.InvalidatePasswordResetToken(req.Token)

	userID := int32(user.ID)
	middleware.AuditAuth(c, h.auditLogger, "PASSWORD_RESET", &userID, "librarian", map[string]interface{}{"email": email, "stage": "completed"})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Password reset successful",
	})
}

// sendPasswordResetEmail sends a password reset email to the user
func (h *AuthHandler) sendPasswordResetEmail(ctx context.Context, user *models.User, token string) error {
	// Get the password reset email template
	template := services.GetDefaultTemplate("password_reset")
	if template == nil {
		return fmt.Errorf("password reset email template not found")
	}

	// Prepare template data
	data := map[string]interface{}{
		"UserName":        getUserDisplayName(user),
		"ResetToken":      token,
		"ExpirationHours": "24", // Token expires in 24 hours (configurable)
	}

	// Send templated email
	return h.emailService.SendTemplatedEmail(ctx, user.Email, template, data)
}

// getUserDisplayName returns a user-friendly display name
func getUserDisplayName(user *models.User) string {
	if user.Username != "" {
		return user.Username
	}
	return user.Email
}
