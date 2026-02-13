package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/middleware"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
)

// InviteHandler handles user invitation endpoints
type InviteHandler struct {
	inviteService *services.InviteService
	authService   *services.AuthService
	baseURL       string
	auditLogger   *middleware.AuditLogger
}

// NewInviteHandler creates a new InviteHandler
func NewInviteHandler(inviteService *services.InviteService, authService *services.AuthService, baseURL string, auditLogger *middleware.AuditLogger) *InviteHandler {
	return &InviteHandler{
		inviteService: inviteService,
		authService:   authService,
		baseURL:       baseURL,
		auditLogger:   auditLogger,
	}
}

// CreateInvite creates a new user invitation
// POST /api/v1/invites
func (h *InviteHandler) CreateInvite(c *gin.Context) {
	var req models.CreateInviteRequest
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

	currentUserID := middleware.GetUserID(c)

	invite, err := h.inviteService.CreateInvite(c.Request.Context(), &req, currentUserID)
	if err != nil {
		if errors.Is(err, services.ErrUserEmailExists) {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "EMAIL_EXISTS",
					"message": "A user with this email already exists",
				},
			})
			return
		}
		if errors.Is(err, services.ErrEmailAlreadyInvited) {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ALREADY_INVITED",
					"message": "An invite has already been sent to this email",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to create invite",
			},
		})
		return
	}

	// Get the token to generate the URL
	token, err := h.inviteService.GetInviteToken(c.Request.Context(), invite.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to get invite token",
			},
		})
		return
	}

	inviteURL := fmt.Sprintf("%s/accept-invite/%s", h.baseURL, token)

	middleware.Audit(c, "invites", int32(invite.ID), "CREATE", nil, map[string]interface{}{"email": invite.Email, "role": invite.Role})
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": models.CreateInviteResponse{
			Invite:    invite,
			InviteURL: inviteURL,
		},
		"message": "Invite created successfully",
	})
}

// ListInvites returns a paginated list of pending invites
// GET /api/v1/invites
func (h *InviteHandler) ListInvites(c *gin.Context) {
	var params models.InviteSearchParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid query parameters",
				"details": err.Error(),
			},
		})
		return
	}

	result, err := h.inviteService.ListPendingInvites(c.Request.Context(), &params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to list invites",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetInvite returns a single invite by ID
// GET /api/v1/invites/:id
func (h *InviteHandler) GetInvite(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "Invalid invite ID",
			},
		})
		return
	}

	invite, err := h.inviteService.GetInviteByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrInviteNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVITE_NOT_FOUND",
					"message": "Invite not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to get invite",
			},
		})
		return
	}

	// Get token if invite is still pending
	var inviteURL string
	if invite.Status == models.InviteStatusPending {
		token, err := h.inviteService.GetInviteToken(c.Request.Context(), id)
		if err == nil {
			inviteURL = fmt.Sprintf("%s/accept-invite/%s", h.baseURL, token)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"invite":     invite,
			"invite_url": inviteURL,
		},
	})
}

// DeleteInvite cancels/deletes an invite
// DELETE /api/v1/invites/:id
func (h *InviteHandler) DeleteInvite(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "Invalid invite ID",
			},
		})
		return
	}

	err = h.inviteService.DeleteInvite(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to delete invite",
			},
		})
		return
	}

	middleware.Audit(c, "invites", int32(id), "DELETE", nil, nil)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Invite deleted successfully",
	})
}

// ResendInvite regenerates the invite token and extends expiry
// POST /api/v1/invites/:id/resend
func (h *InviteHandler) ResendInvite(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "Invalid invite ID",
			},
		})
		return
	}

	invite, err := h.inviteService.ResendInvite(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrInviteNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVITE_NOT_FOUND",
					"message": "Invite not found",
				},
			})
			return
		}
		if errors.Is(err, services.ErrInviteAlreadyUsed) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVITE_ALREADY_USED",
					"message": "This invite has already been accepted",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to resend invite",
			},
		})
		return
	}

	// Get the new token to generate the URL
	token, err := h.inviteService.GetInviteToken(c.Request.Context(), invite.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to get invite token",
			},
		})
		return
	}

	inviteURL := fmt.Sprintf("%s/accept-invite/%s", h.baseURL, token)

	middleware.Audit(c, "invites", int32(invite.ID), "UPDATE", nil, map[string]interface{}{"action": "resend", "email": invite.Email})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": models.CreateInviteResponse{
			Invite:    invite,
			InviteURL: inviteURL,
		},
		"message": "Invite resent successfully",
	})
}

// ValidateInvite validates an invite token (public endpoint)
// GET /api/v1/auth/invite/:token
func (h *InviteHandler) ValidateInvite(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_TOKEN",
				"message": "Token is required",
			},
		})
		return
	}

	response, err := h.inviteService.ValidateInviteToken(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to validate token",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// AcceptInvite accepts an invitation and creates the user account
// POST /api/v1/auth/invite/accept
func (h *InviteHandler) AcceptInvite(c *gin.Context) {
	var req models.AcceptInviteRequest
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

	// Debug logging for token received
	slog.Info("AcceptInvite request received",
		"token_length", len(req.Token),
		"token_first_10", req.Token[:min(10, len(req.Token))],
		"username", req.Username,
	)

	// Validate passwords match
	if req.Password != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "PASSWORD_MISMATCH",
				"message": "Passwords do not match",
			},
		})
		return
	}

	// Hash the password
	hashedPassword, err := h.authService.HashPassword(req.Password)
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

	user, err := h.inviteService.AcceptInvite(c.Request.Context(), &req, hashedPassword)
	if err != nil {
		if errors.Is(err, services.ErrInviteNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_TOKEN",
					"message": "Invalid invite token",
				},
			})
			return
		}
		if errors.Is(err, services.ErrInviteExpired) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVITE_EXPIRED",
					"message": "This invite has expired",
				},
			})
			return
		}
		if errors.Is(err, services.ErrInviteAlreadyUsed) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVITE_ALREADY_USED",
					"message": "This invite has already been accepted",
				},
			})
			return
		}
		if errors.Is(err, services.ErrUsernameExists) {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "USERNAME_EXISTS",
					"message": "Username already exists",
				},
			})
			return
		}
		if errors.Is(err, services.ErrUserEmailExists) {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "EMAIL_EXISTS",
					"message": "A user with this email already exists",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to accept invite",
			},
		})
		return
	}

	userID := int32(user.ID)
	middleware.AuditAuth(c, h.auditLogger, "CREATE", &userID, "system", map[string]interface{}{"action": "accept_invite", "username": user.Username, "email": user.Email, "role": user.Role})
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    user.ToResponse(),
		"message": "Account created successfully. You can now log in.",
	})
}
