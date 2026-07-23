package controllers

import (
	"log/slog"
	"net/http"

	"backend/dto"
	"backend/services"

	"github.com/gin-gonic/gin"
)

type AdminDashboardController struct {
	superAdminSvc services.SuperAdminService
}

func NewAdminDashboardController(superAdminSvc services.SuperAdminService) *AdminDashboardController {
	return &AdminDashboardController{superAdminSvc: superAdminSvc}
}

func (ctrl *AdminDashboardController) GetDashboard(c *gin.Context) {
	stats, err := ctrl.superAdminSvc.GetDashboardStats(c.Request.Context())
	if err != nil {
		slog.Error("Failed to fetch admin dashboard stats", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to retrieve dashboard statistics.",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Dashboard statistics retrieved successfully.",
		Data:    stats,
	})
}
