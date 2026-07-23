package controllers

import (
	"errors"
	"log/slog"
	"net/http"

	"backend/dto"
	"backend/services"

	"github.com/gin-gonic/gin"
)

type SuperAdminController struct {
	superAdminSvc services.SuperAdminService
	env           string
}

func NewSuperAdminController(superAdminSvc services.SuperAdminService, env string) *SuperAdminController {
	return &SuperAdminController{
		superAdminSvc: superAdminSvc,
		env:           env,
	}
}

// ── Dashboard ──

func (sc *SuperAdminController) GetDashboard(c *gin.Context) {
	stats, err := sc.superAdminSvc.GetDashboardStats(c.Request.Context())
	if err != nil {
		slog.Error("Failed to fetch dashboard stats", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to retrieve dashboard statistics.",
			Error:   dto.ErrorDetail{Code: "INTERNAL_ERROR"},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Dashboard statistics retrieved successfully.",
		Data:    stats,
	})
}

// ── List Admins ──

func (sc *SuperAdminController) ListAdmins(c *gin.Context) {
	var query dto.AdminListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: "Invalid query parameters.",
			Error:   dto.ErrorDetail{Code: "VALIDATION_ERROR"},
		})
		return
	}

	result, err := sc.superAdminSvc.ListAdmins(c.Request.Context(), query)
	if err != nil {
		slog.Error("Failed to list admins", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to retrieve admins.",
			Error:   dto.ErrorDetail{Code: "INTERNAL_ERROR"},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    result,
	})
}

// ── Create Admin ──

func (sc *SuperAdminController) CreateAdmin(c *gin.Context) {
	var req dto.CreateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: "Email, password, first_name, and last_name are required.",
			Error:   dto.ErrorDetail{Code: "VALIDATION_ERROR"},
		})
		return
	}

	user, err := sc.superAdminSvc.CreateAdmin(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, services.ErrEmailAlreadyExists) {
			c.JSON(http.StatusConflict, dto.ErrorResponse{
				Success: false,
				Message: "An account with this email already exists.",
				Error:   dto.ErrorDetail{Code: "EMAIL_TAKEN"},
			})
			return
		}
		if isPasswordValidationError(err) {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: "WEAK_PASSWORD"},
			})
			return
		}
		slog.Error("Failed to create admin", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to create admin account.",
			Error:   dto.ErrorDetail{Code: "INTERNAL_ERROR"},
		})
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse{
		Success: true,
		Message: "Admin account created successfully.",
		Data: gin.H{
			"admin": gin.H{
				"id":         user.ID,
				"email":      user.Email,
				"first_name": user.FirstName,
				"last_name":  user.LastName,
				"role":       user.Role,
				"phone":      user.Phone,
				"is_active":  user.IsActive,
			},
		},
	})
}

// ── Get Admin by ID ──

func (sc *SuperAdminController) GetAdmin(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: "Admin ID is required.",
			Error:   dto.ErrorDetail{Code: "VALIDATION_ERROR"},
		})
		return
	}

	user, err := sc.superAdminSvc.GetAdminByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrAdminNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Success: false,
				Message: "Admin not found.",
				Error:   dto.ErrorDetail{Code: "ADMIN_NOT_FOUND"},
			})
			return
		}
		slog.Error("Failed to get admin", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to retrieve admin.",
			Error:   dto.ErrorDetail{Code: "INTERNAL_ERROR"},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data: gin.H{
			"admin": gin.H{
				"id":         user.ID,
				"email":      user.Email,
				"first_name": user.FirstName,
				"last_name":  user.LastName,
				"role":       user.Role,
				"phone":      user.Phone,
				"is_active":  user.IsActive,
				"created_at": user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				"updated_at": user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			},
		},
	})
}

// ── Update Admin ──

func (sc *SuperAdminController) UpdateAdmin(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: "Admin ID is required.",
			Error:   dto.ErrorDetail{Code: "VALIDATION_ERROR"},
		})
		return
	}

	var req dto.UpdateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: "Invalid request body.",
			Error:   dto.ErrorDetail{Code: "VALIDATION_ERROR"},
		})
		return
	}

	user, err := sc.superAdminSvc.UpdateAdmin(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, services.ErrAdminNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Success: false,
				Message: "Admin not found.",
				Error:   dto.ErrorDetail{Code: "ADMIN_NOT_FOUND"},
			})
			return
		}
		if errors.Is(err, services.ErrEmailAlreadyExists) {
			c.JSON(http.StatusConflict, dto.ErrorResponse{
				Success: false,
				Message: "An account with this email already exists.",
				Error:   dto.ErrorDetail{Code: "EMAIL_TAKEN"},
			})
			return
		}
		slog.Error("Failed to update admin", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to update admin.",
			Error:   dto.ErrorDetail{Code: "INTERNAL_ERROR"},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Admin updated successfully.",
		Data: gin.H{
			"admin": gin.H{
				"id":         user.ID,
				"email":      user.Email,
				"first_name": user.FirstName,
				"last_name":  user.LastName,
				"role":       user.Role,
				"phone":      user.Phone,
				"is_active":  user.IsActive,
				"created_at": user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				"updated_at": user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			},
		},
	})
}

// ── Update Admin Status ──

func (sc *SuperAdminController) UpdateAdminStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: "Admin ID is required.",
			Error:   dto.ErrorDetail{Code: "VALIDATION_ERROR"},
		})
		return
	}

	var req dto.UpdateAdminStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: "Invalid request body. is_active is required.",
			Error:   dto.ErrorDetail{Code: "VALIDATION_ERROR"},
		})
		return
	}

	if err := sc.superAdminSvc.UpdateAdminStatus(c.Request.Context(), id, req); err != nil {
		if errors.Is(err, services.ErrAdminNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Success: false,
				Message: "Admin not found.",
				Error:   dto.ErrorDetail{Code: "ADMIN_NOT_FOUND"},
			})
			return
		}
		slog.Error("Failed to update admin status", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to update admin status.",
			Error:   dto.ErrorDetail{Code: "INTERNAL_ERROR"},
		})
		return
	}

	statusText := "activated"
	if !req.IsActive {
		statusText = "deactivated"
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Admin " + statusText + " successfully.",
	})
}

// ── Delete Admin ──

func (sc *SuperAdminController) DeleteAdmin(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: "Admin ID is required.",
			Error:   dto.ErrorDetail{Code: "VALIDATION_ERROR"},
		})
		return
	}

	currentUserId, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Success: false,
			Message: "Authentication required.",
			Error:   dto.ErrorDetail{Code: "UNAUTHORIZED"},
		})
		return
	}

	if err := sc.superAdminSvc.DeleteAdmin(c.Request.Context(), id, currentUserId.(string)); err != nil {
		if errors.Is(err, services.ErrAdminNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Success: false,
				Message: "Admin not found.",
				Error:   dto.ErrorDetail{Code: "ADMIN_NOT_FOUND"},
			})
			return
		}
		if errors.Is(err, services.ErrCannotDeleteSelf) {
			c.JSON(http.StatusConflict, dto.ErrorResponse{
				Success: false,
				Message: "You cannot delete your own account.",
				Error:   dto.ErrorDetail{Code: "CANNOT_DELETE_SELF"},
			})
			return
		}
		if errors.Is(err, services.ErrCannotDeleteSuper) {
			c.JSON(http.StatusForbidden, dto.ErrorResponse{
				Success: false,
				Message: "Cannot delete another super admin.",
				Error:   dto.ErrorDetail{Code: "FORBIDDEN"},
			})
			return
		}
		slog.Error("Failed to delete admin", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to delete admin.",
			Error:   dto.ErrorDetail{Code: "INTERNAL_ERROR"},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Admin deleted successfully.",
	})
}
