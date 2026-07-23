package controllers

import (
	"log/slog"
	"net/http"
	"strconv"

	"backend/dto"
	"backend/services"

	"github.com/gin-gonic/gin"
)

type CustomerController struct {
	customerSvc services.CustomerService
}

func NewCustomerController(customerSvc services.CustomerService) *CustomerController {
	return &CustomerController{customerSvc: customerSvc}
}

func (ctrl *CustomerController) ListCustomers(c *gin.Context) {
	search := c.DefaultQuery("search", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	result, err := ctrl.customerSvc.List(c.Request.Context(), search, page, limit)
	if err != nil {
		slog.Error("failed to list customers", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to list customers",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Customers fetched successfully",
		Data:    result,
	})
}
