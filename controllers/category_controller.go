package controllers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"backend/dto"
	"backend/services"

	"github.com/gin-gonic/gin"
)

type CategoryController struct {
	categorySvc services.CategoryService
}

func NewCategoryController(categorySvc services.CategoryService) *CategoryController {
	return &CategoryController{categorySvc: categorySvc}
}

func (ctrl *CategoryController) CreateCategory(c *gin.Context) {
	var req dto.CreateCategoryRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: err.Error(),
			Error:   dto.ErrorDetail{Code: dto.VALIDATION_ERROR},
		})
		return
	}

	category, err := ctrl.categorySvc.Create(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, services.ErrCategoryNotFound) || errors.Is(err, services.ErrInvalidParent) {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: dto.INVALID_CATEGORY},
			})
			return
		}

		slog.Error("failed to create category", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to create category",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse{
		Success: true,
		Message: "Category created successfully",
		Data:    category,
	})
}

func (ctrl *CategoryController) GetCategory(c *gin.Context) {
	id := c.Param("id")

	category, err := ctrl.categorySvc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrCategoryNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: dto.CATEGORY_NOT_FOUND},
			})
			return
		}

		slog.Error("failed to get category", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to get category",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Category fetched successfully",
		Data:    category,
	})
}

func (ctrl *CategoryController) ListCategories(c *gin.Context) {
	search := c.DefaultQuery("search", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, err := ctrl.categorySvc.List(c.Request.Context(), search, page, limit)
	if err != nil {
		slog.Error("failed to list categories", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to list categories",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Categories fetched successfully",
		Data:    result,
	})
}

func (ctrl *CategoryController) ListActiveCategories(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, err := ctrl.categorySvc.ListActive(c.Request.Context(), page, limit)
	if err != nil {
		slog.Error("failed to list active categories", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to list active categories",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Active categories fetched successfully",
		Data:    result,
	})
}

func (ctrl *CategoryController) UpdateCategory(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: err.Error(),
			Error:   dto.ErrorDetail{Code: dto.VALIDATION_ERROR},
		})
		return
	}

	category, err := ctrl.categorySvc.Update(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, services.ErrCategoryNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: dto.CATEGORY_NOT_FOUND},
			})
			return
		}

		slog.Error("failed to update category", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to update category",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Category updated successfully",
		Data:    category,
	})
}

func (ctrl *CategoryController) DeleteCategory(c *gin.Context) {
	id := c.Param("id")

	err := ctrl.categorySvc.Delete(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrCategoryNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: dto.CATEGORY_NOT_FOUND},
			})
			return
		}

		if errors.Is(err, services.ErrCategoryHasProducts) {
			c.JSON(http.StatusConflict, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: dto.CATEGORY_HAS_PRODUCTS},
			})
			return
		}

		slog.Error("failed to delete category", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to delete category",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Category deleted successfully",
	})
}
