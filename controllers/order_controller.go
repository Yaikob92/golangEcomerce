package controllers

import (
	"log/slog"
	"net/http"
	"strconv"

	"backend/dto"
	"backend/services"

	"github.com/gin-gonic/gin"
)

type OrderController struct {
	orderSvc services.OrderService
}

func NewOrderController(orderSvc services.OrderService) *OrderController {
	return &OrderController{orderSvc: orderSvc}
}

func (ctrl *OrderController) ListOrders(c *gin.Context) {
	search := c.DefaultQuery("search", "")
	status := c.DefaultQuery("status", "")
	sort := c.DefaultQuery("sort", "-created_at")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	result, err := ctrl.orderSvc.List(c.Request.Context(), search, status, sort, page, limit)
	if err != nil {
		slog.Error("failed to list orders", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to list orders",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Orders fetched successfully",
		Data:    result,
	})
}

func (ctrl *OrderController) GetOrder(c *gin.Context) {
	id := c.Param("id")

	order, err := ctrl.orderSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Success: false,
			Message: err.Error(),
			Error:   dto.ErrorDetail{Code: dto.NOT_FOUND},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Order fetched successfully",
		Data:    order,
	})
}
