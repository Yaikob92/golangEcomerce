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

type BrandController struct {
	brandSvc services.BrandService
}

func NewBrandController(brandSvc services.BrandService) *BrandController {
	return &BrandController{brandSvc: brandSvc}
}

func (ctrl *BrandController) CreateBrand(c *gin.Context) {
	var req dto.CreateBrandRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: err.Error(),
			Error:   dto.ErrorDetail{Code: dto.VALIDATION_ERROR},
		})
		return
	}

	brand, err := ctrl.brandSvc.Create(c.Request.Context(), req)
	if err != nil {
		slog.Error("failed to create brand", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to create brand",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse{
		Success: true,
		Message: "Brand created successfully",
		Data:    brand,
	})
}

func (ctrl *BrandController) GetBrand(c *gin.Context) {
	id := c.Param("id")

	brand, err := ctrl.brandSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrBrandNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: dto.BRAND_NOT_FOUND},
			})
			return
		}

		slog.Error("failed to get brand", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to get brand",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Brand fetched successfully",
		Data:    brand,
	})
}

func (ctrl *BrandController) ListBrands(c *gin.Context) {
	search := c.DefaultQuery("search", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, err := ctrl.brandSvc.List(c.Request.Context(), search, page, limit)
	if err != nil {
		slog.Error("failed to list brands", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to list brands",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Brands fetched successfully",
		Data:    result,
	})
}

func (ctrl *BrandController) ListActiveBrands(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, err := ctrl.brandSvc.ListActive(c.Request.Context(), page, limit)
	if err != nil {
		slog.Error("failed to list active brands", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to list active brands",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Active brands fetched successfully",
		Data:    result,
	})
}

func (ctrl *BrandController) UpdateBrand(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateBrandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: err.Error(),
			Error:   dto.ErrorDetail{Code: dto.VALIDATION_ERROR},
		})
		return
	}

	brand, err := ctrl.brandSvc.Update(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, services.ErrBrandNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: dto.BRAND_NOT_FOUND},
			})
			return
		}

		slog.Error("failed to update brand", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to update brand",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Brand updated successfully",
		Data:    brand,
	})
}

func (ctrl *BrandController) DeleteBrand(c *gin.Context) {
	id := c.Param("id")

	err := ctrl.brandSvc.Delete(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrBrandNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: dto.BRAND_NOT_FOUND},
			})
			return
		}

		if errors.Is(err, services.ErrBrandHasProducts) {
			c.JSON(http.StatusConflict, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: dto.BRAND_HAS_PRODUCTS},
			})
			return
		}

		slog.Error("failed to delete brand", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to delete brand",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Brand deleted successfully",
	})
}
