package controllers

import (
	"errors"
	"log/slog"
	"net/http"

	"backend/dto"
	"backend/services"

	"github.com/gin-gonic/gin"
)

type ProductController struct {
	productSvc      services.ProductService
	productImageSvc services.ProductImageService
}

func NewProductController(productSvc services.ProductService, productImageSvc services.ProductImageService) *ProductController {
	return &ProductController{
		productSvc:      productSvc,
		productImageSvc: productImageSvc,
	}
}

func (ctrl *ProductController) CreateProduct(c *gin.Context) {
	var req dto.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: err.Error(),
			Error:   dto.ErrorDetail{Code: dto.VALIDATION_ERROR},
		})
		return
	}

	product, err := ctrl.productSvc.Create(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCategory) || errors.Is(err, services.ErrInvalidBrand) || errors.Is(err, services.ErrDiscountExceedsPrice) {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: dto.VALIDATION_ERROR},
			})
			return
		}
		if errors.Is(err, services.ErrProductSlugExists) {
			c.JSON(http.StatusConflict, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: dto.SLUG_EXISTS},
			})
			return
		}
		if errors.Is(err, services.ErrProductSKUExists) {
			c.JSON(http.StatusConflict, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: dto.SKU_EXISTS},
			})
			return
		}

		slog.Error("failed to create product", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to create product",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse{
		Success: true,
		Message: "Product created successfully",
		Data:    product,
	})
}

func (ctrl *ProductController) GetProduct(c *gin.Context) {
	id := c.Param("id")

	product, err := ctrl.productSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrProductNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: dto.PRODUCT_NOT_FOUND},
			})
			return
		}

		slog.Error("failed to get product", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to get product",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Product fetched successfully",
		Data:    product,
	})
}

func (ctrl *ProductController) GetProductBySlug(c *gin.Context) {
	slug := c.Param("slug")

	product, err := ctrl.productSvc.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, services.ErrProductNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: dto.PRODUCT_NOT_FOUND},
			})
			return
		}

		slog.Error("failed to get product by slug", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to get product",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Product fetched successfully",
		Data:    product,
	})
}

func (ctrl *ProductController) ListProducts(c *gin.Context) {
	var query dto.ProductListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: err.Error(),
			Error:   dto.ErrorDetail{Code: dto.VALIDATION_ERROR},
		})
		return
	}
	query.SetDefaults()

	result, err := ctrl.productSvc.List(c.Request.Context(), query)
	if err != nil {
		slog.Error("failed to list products", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to list products",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Products fetched successfully",
		Data:    result,
	})
}

func (ctrl *ProductController) ListPublicProducts(c *gin.Context) {
	var query dto.ProductListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: err.Error(),
			Error:   dto.ErrorDetail{Code: dto.VALIDATION_ERROR},
		})
		return
	}
	query.SetDefaults()

	result, err := ctrl.productSvc.ListPublic(c.Request.Context(), query)
	if err != nil {
		slog.Error("failed to list public products", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to list products",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Products fetched successfully",
		Data:    result,
	})
}

func (ctrl *ProductController) UpdateProduct(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: err.Error(),
			Error:   dto.ErrorDetail{Code: dto.VALIDATION_ERROR},
		})
		return
	}

	product, err := ctrl.productSvc.Update(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, services.ErrProductNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: dto.PRODUCT_NOT_FOUND},
			})
			return
		}
		if errors.Is(err, services.ErrInvalidCategory) || errors.Is(err, services.ErrInvalidBrand) || errors.Is(err, services.ErrDiscountExceedsPrice) {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: dto.VALIDATION_ERROR},
			})
			return
		}

		slog.Error("failed to update product", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to update product",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Product updated successfully",
		Data:    product,
	})
}

func (ctrl *ProductController) DeleteProduct(c *gin.Context) {
	id := c.Param("id")

	err := ctrl.productSvc.Delete(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrProductNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: dto.PRODUCT_NOT_FOUND},
			})
			return
		}

		slog.Error("failed to delete product", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to delete product",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Product deleted successfully",
	})
}

// ── Product Images ──

func (ctrl *ProductController) GetProductImages(c *gin.Context) {
	id := c.Param("id")

	images, err := ctrl.productImageSvc.GetImages(c.Request.Context(), id)
	if err != nil {
		slog.Error("failed to get product images", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to get product images",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Product images fetched successfully",
		Data:    images,
	})
}

func (ctrl *ProductController) AddProductImage(c *gin.Context) {
	id := c.Param("id")

	var req dto.AddProductImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: err.Error(),
			Error:   dto.ErrorDetail{Code: dto.VALIDATION_ERROR},
		})
		return
	}

	image, err := ctrl.productImageSvc.AddImage(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, services.ErrImageLimitReached) {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: dto.IMAGE_LIMIT_REACHED},
			})
			return
		}
		if errors.Is(err, services.ErrProductNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: dto.PRODUCT_NOT_FOUND},
			})
			return
		}

		slog.Error("failed to add product image", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to add product image",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse{
		Success: true,
		Message: "Product image added successfully",
		Data:    image,
	})
}

func (ctrl *ProductController) RemoveProductImage(c *gin.Context) {
	imageId := c.Param("imageId")

	err := ctrl.productImageSvc.RemoveImage(c.Request.Context(), imageId)
	if err != nil {
		if errors.Is(err, services.ErrImageNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: dto.IMAGE_NOT_FOUND},
			})
			return
		}

		slog.Error("failed to remove product image", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to remove product image",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Image removed successfully",
	})
}

func (ctrl *ProductController) SetPrimaryImage(c *gin.Context) {
	id := c.Param("id")
	imageId := c.Param("imageId")

	err := ctrl.productImageSvc.SetPrimary(c.Request.Context(), id, imageId)
	if err != nil {
		if errors.Is(err, services.ErrImageNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: dto.IMAGE_NOT_FOUND},
			})
			return
		}

		slog.Error("failed to set primary image", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to set primary image",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Primary image updated",
	})
}

func (ctrl *ProductController) ReorderImages(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateImageOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: err.Error(),
			Error:   dto.ErrorDetail{Code: dto.VALIDATION_ERROR},
		})
		return
	}

	err := ctrl.productImageSvc.ReorderImages(c.Request.Context(), id, req)
	if err != nil {
		slog.Error("failed to reorder images", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to reorder images",
			Error:   dto.ErrorDetail{Code: dto.INTERNAL_ERROR},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Images reordered",
	})
}
