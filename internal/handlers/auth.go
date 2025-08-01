package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/middleware"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
)

type AuthHandler struct {
	authService *services.AuthService
	userService services.UserServiceInterface
}

func NewAuthHandler(authService *services.AuthService, userService services.UserServiceInterface) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		userService: userService,
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
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

	// Try to authenticate as librarian first
	user, err := h.userService.GetUserByUsername(req.Username)
	if err == nil && user != nil {
		// Verify password
		isValid, err := h.authService.VerifyPassword(user.PasswordHash, req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "Error verifying password",
				},
			})
			return

		}

		if !isValid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_CREDENTIALS",
					"message": "Invalid username or password",
				},
			})
			return
		}

		if !user.IsActive {
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
		accessToken, refreshToken, err := h.authService.GenerateTokens(user, "librarian")
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

		// Update last login
		err = h.userService.UpdateLastLogin(user.ID)
		if err != nil {
			// Log error but don't fail the login
		}

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
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_CREDENTIALS",
				"message": "Invalid username or password",
			},
		})
		return
	}

	// For students, if no password is set, use student ID as default password
	if student.PasswordHash == nil {
		if req.Password != student.StudentID {
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
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "Error verifying password",
				},
			})
			return
		}

		if !isValid {
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

func (h *AuthHandler) Logout(c *gin.Context) {
	// Get token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Logout successful",
		})
		return
	}

	// Extract token from Bearer header
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Logout successful",
		})
		return
	}

	tokenString := parts[1]

	// Blacklist the access token
	if err := h.authService.BlacklistToken(tokenString); err != nil {
		// Log error but don't fail the logout
		// User should still be logged out on the client side
	}

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
	_, err := h.userService.GetUserByEmail(req.Email)
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

	// In a real application, you would send this token via email
	// For now, we'll just log it (DON'T DO THIS IN PRODUCTION)
	// You should integrate with an email service like SendGrid, SES, etc.
	_ = token // TODO: Send via email

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

	// Invalidate the reset token
	err = h.authService.InvalidatePasswordResetToken(req.Token)
	if err != nil {
		// Log error but don't fail the request
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Password reset successful",
	})
}
