package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"backend/dto"
	"backend/models"
	"backend/repositories"
	"backend/utils"
)

var (
	ErrProductNotFound      = errors.New("product not found")
	ErrInvalidCategory      = errors.New("invalid category")
	ErrInvalidBrand         = errors.New("invalid brand")
	ErrDiscountExceedsPrice = errors.New("discount price cannot exceed regular price")
	ErrProductSlugExists    = errors.New("product slug already exists")
	ErrProductSKUExists     = errors.New("product SKU already exists")
)

type ProductService interface {
	Create(ctx context.Context, req dto.CreateProductRequest) (dto.ProductResponse, error)
	GetByID(ctx context.Context, id string) (dto.ProductResponse, error)
	GetBySlug(ctx context.Context, slug string) (dto.ProductResponse, error)
	List(ctx context.Context, query dto.ProductListQuery) (dto.ProductListResponse, error)
	ListPublic(ctx context.Context, query dto.ProductListQuery) (dto.ProductListResponse, error)
	Update(ctx context.Context, id string, req dto.UpdateProductRequest) (dto.ProductResponse, error)
	Delete(ctx context.Context, id string) error
}

type productService struct {
	productRepo      repositories.ProductRepository
	productImageRepo repositories.ProductImageRepository
	categoryRepo     repositories.CategoryRepository
	brandRepo        repositories.BrandRepository
}

func NewProductService(
	productRepo repositories.ProductRepository,
	productImageRepo repositories.ProductImageRepository,
	categoryRepo repositories.CategoryRepository,
	brandRepo repositories.BrandRepository,
) ProductService {
	return &productService{
		productRepo:      productRepo,
		productImageRepo: productImageRepo,
		categoryRepo:     categoryRepo,
		brandRepo:        brandRepo,
	}
}

func dtoToRepoQuery(q dto.ProductListQuery) repositories.ProductListQuery {
	offset := 0
	if q.Page > 1 {
		offset = (q.Page - 1) * q.Limit
	}
	return repositories.ProductListQuery{
		Search:     q.Search,
		CategoryID: q.CategoryID,
		BrandID:    q.BrandID,
		Status:     q.Status,
		IsFeatured: q.IsFeatured,
		MinPrice:   q.MinPrice,
		MaxPrice:   q.MaxPrice,
		Sort:       q.Sort,
		Offset:     offset,
		Limit:      q.Limit,
	}
}

func (s *productService) Create(ctx context.Context, req dto.CreateProductRequest) (dto.ProductResponse, error) {
	_, err := s.categoryRepo.GetByID(ctx, req.CategoryID)
	if err != nil {
		slog.Error("category not found for product creation", "category_id", req.CategoryID, "error", err)
		return dto.ProductResponse{}, ErrInvalidCategory
	}

	var brandID *string
	if req.BrandID != "" {
		_, err := s.brandRepo.GetByID(ctx, req.BrandID)
		if err != nil {
			slog.Error("brand not found for product creation", "brand_id", req.BrandID, "error", err)
			return dto.ProductResponse{}, ErrInvalidBrand
		}
		brandID = &req.BrandID
	}

	if req.DiscountPrice != nil && *req.DiscountPrice >= req.Price {
		return dto.ProductResponse{}, ErrDiscountExceedsPrice
	}

	slug := utils.GenerateSlug(req.Name)

	sku := req.SKU
	if sku == "" {
		sku = utils.GenerateProductSKU(req.Name)
	}

	status := req.Status
	if status == "" {
		status = "active"
	}

	product := models.Product{
		Name:            req.Name,
		Slug:            slug,
		Description:     req.Description,
		Price:           req.Price,
		DiscountPrice:   req.DiscountPrice,
		SKU:             sku,
		StockQuantity:   req.StockQuantity,
		CategoryID:      req.CategoryID,
		BrandID:         brandID,
		Status:          status,
		IsFeatured:      req.IsFeatured,
		Weight:          req.Weight,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
	}

	created, err := s.productRepo.Create(ctx, product)
	if err != nil {
		if errors.Is(err, repositories.ErrProductSlugExists) {
			slug = slug + "-" + utils.GenerateShortID()
			product.Slug = slug
			created, err = s.productRepo.Create(ctx, product)
		}
		if errors.Is(err, repositories.ErrProductSKUExists) {
			return dto.ProductResponse{}, ErrProductSKUExists
		}
		if err != nil {
			slog.Error("failed to create product", "error", err)
			return dto.ProductResponse{}, fmt.Errorf("failed to create product: %w", err)
		}
	}

	images, err := s.productImageRepo.GetByProductID(ctx, created.ID)
	if err != nil {
		slog.Warn("failed to fetch product images after creation", "product_id", created.ID, "error", err)
		images = []models.ProductImage{}
	}

	return toProductResponse(created, images), nil
}

func (s *productService) GetByID(ctx context.Context, id string) (dto.ProductResponse, error) {
	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		slog.Error("product not found", "id", id, "error", err)
		return dto.ProductResponse{}, ErrProductNotFound
	}

	images, err := s.productImageRepo.GetByProductID(ctx, product.ID)
	if err != nil {
		slog.Warn("failed to fetch product images", "product_id", product.ID, "error", err)
		images = []models.ProductImage{}
	}

	return toProductResponse(product, images), nil
}

func (s *productService) GetBySlug(ctx context.Context, slug string) (dto.ProductResponse, error) {
	product, err := s.productRepo.GetBySlug(ctx, slug)
	if err != nil {
		slog.Error("product not found by slug", "slug", slug, "error", err)
		return dto.ProductResponse{}, ErrProductNotFound
	}

	images, err := s.productImageRepo.GetByProductID(ctx, product.ID)
	if err != nil {
		slog.Warn("failed to fetch product images", "product_id", product.ID, "error", err)
		images = []models.ProductImage{}
	}

	return toProductResponse(product, images), nil
}

func (s *productService) List(ctx context.Context, query dto.ProductListQuery) (dto.ProductListResponse, error) {
	repoQuery := dtoToRepoQuery(query)
	products, total, err := s.productRepo.List(ctx, repoQuery)
	if err != nil {
		slog.Error("failed to list products", "error", err)
		return dto.ProductListResponse{}, fmt.Errorf("failed to list products: %w", err)
	}

	var productResponses []dto.ProductResponse
	for _, p := range products {
		images, err := s.productImageRepo.GetByProductID(ctx, p.ID)
		if err != nil {
			slog.Warn("failed to fetch product images", "product_id", p.ID, "error", err)
			images = []models.ProductImage{}
		}
		productResponses = append(productResponses, toProductResponse(p, images))
	}

	totalPages := 1
	if query.Limit > 0 {
		totalPages = total / query.Limit
		if total%query.Limit != 0 {
			totalPages++
		}
	}

	return dto.ProductListResponse{
		Products:   productResponses,
		TotalCount: total,
		Page:       query.Page,
		Limit:      query.Limit,
		TotalPages: totalPages,
	}, nil
}

func (s *productService) ListPublic(ctx context.Context, query dto.ProductListQuery) (dto.ProductListResponse, error) {
	repoQuery := dtoToRepoQuery(query)
	products, total, err := s.productRepo.ListPublic(ctx, repoQuery)
	if err != nil {
		slog.Error("failed to list public products", "error", err)
		return dto.ProductListResponse{}, fmt.Errorf("failed to list public products: %w", err)
	}

	var productResponses []dto.ProductResponse
	for _, p := range products {
		images, err := s.productImageRepo.GetByProductID(ctx, p.ID)
		if err != nil {
			slog.Warn("failed to fetch product images", "product_id", p.ID, "error", err)
			images = []models.ProductImage{}
		}
		productResponses = append(productResponses, toProductResponse(p, images))
	}

	totalPages := 1
	if query.Limit > 0 {
		totalPages = total / query.Limit
		if total%query.Limit != 0 {
			totalPages++
		}
	}

	return dto.ProductListResponse{
		Products:   productResponses,
		TotalCount: total,
		Page:       query.Page,
		Limit:      query.Limit,
		TotalPages: totalPages,
	}, nil
}

func (s *productService) Update(ctx context.Context, id string, req dto.UpdateProductRequest) (dto.ProductResponse, error) {
	existing, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		slog.Error("product not found for update", "id", id, "error", err)
		return dto.ProductResponse{}, ErrProductNotFound
	}

	if req.CategoryID != "" && req.CategoryID != existing.CategoryID {
		_, err := s.categoryRepo.GetByID(ctx, req.CategoryID)
		if err != nil {
			slog.Error("category not found for product update", "category_id", req.CategoryID, "error", err)
			return dto.ProductResponse{}, ErrInvalidCategory
		}
		existing.CategoryID = req.CategoryID
	}

	if req.BrandID != "" {
		var newBrandID *string
		if existing.BrandID == nil || req.BrandID != *existing.BrandID {
			_, err := s.brandRepo.GetByID(ctx, req.BrandID)
			if err != nil {
				slog.Error("brand not found for product update", "brand_id", req.BrandID, "error", err)
				return dto.ProductResponse{}, ErrInvalidBrand
			}
			newBrandID = &req.BrandID
			existing.BrandID = newBrandID
		}
	}

	if req.Name != "" {
		existing.Name = req.Name
		existing.Slug = utils.GenerateSlug(req.Name)
	}

	if req.Description != "" {
		existing.Description = req.Description
	}

	if req.Price != nil && *req.Price > 0 {
		existing.Price = *req.Price
	}

	if req.DiscountPrice != nil {
		existing.DiscountPrice = req.DiscountPrice
	}

	if req.DiscountPrice != nil && existing.DiscountPrice != nil && *existing.DiscountPrice >= existing.Price {
		return dto.ProductResponse{}, ErrDiscountExceedsPrice
	}

	if req.SKU != "" {
		existing.SKU = req.SKU
	}

	if req.StockQuantity != nil && *req.StockQuantity >= 0 {
		existing.StockQuantity = *req.StockQuantity
	}

	if req.Status != "" {
		existing.Status = req.Status
	}

	if req.IsFeatured != nil {
		existing.IsFeatured = *req.IsFeatured
	}

	if req.Weight != nil {
		existing.Weight = req.Weight
	}

	if req.MetaTitle != "" {
		existing.MetaTitle = req.MetaTitle
	}

	if req.MetaDescription != "" {
		existing.MetaDescription = req.MetaDescription
	}

	updated, err := s.productRepo.Update(ctx, id, existing)
	if err != nil {
		if errors.Is(err, repositories.ErrProductSlugExists) {
			return dto.ProductResponse{}, ErrProductSlugExists
		}
		if errors.Is(err, repositories.ErrProductSKUExists) {
			return dto.ProductResponse{}, ErrProductSKUExists
		}
		slog.Error("failed to update product", "id", id, "error", err)
		return dto.ProductResponse{}, fmt.Errorf("failed to update product: %w", err)
	}

	images, err := s.productImageRepo.GetByProductID(ctx, updated.ID)
	if err != nil {
		slog.Warn("failed to fetch product images after update", "product_id", updated.ID, "error", err)
		images = []models.ProductImage{}
	}

	return toProductResponse(updated, images), nil
}

func (s *productService) Delete(ctx context.Context, id string) error {
	_, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		slog.Error("product not found for deletion", "id", id, "error", err)
		return ErrProductNotFound
	}

	if err := s.productRepo.SoftDelete(ctx, id); err != nil {
		slog.Error("failed to delete product", "id", id, "error", err)
		return fmt.Errorf("failed to delete product: %w", err)
	}

	return nil
}

func toProductResponse(p models.Product, images []models.ProductImage) dto.ProductResponse {
	var imageResponses []dto.ProductImageResponse
	for _, img := range images {
		imageResponses = append(imageResponses, toProductImageResponse(img))
	}

	return dto.ProductResponse{
		ID:              p.ID,
		Name:            p.Name,
		Slug:            p.Slug,
		Description:     p.Description,
		SKU:             p.SKU,
		Price:           p.Price,
		DiscountPrice:   p.DiscountPrice,
		StockQuantity:   p.StockQuantity,
		CategoryID:      p.CategoryID,
		BrandID:         p.BrandID,
		Status:          p.Status,
		IsFeatured:      p.IsFeatured,
		Weight:          p.Weight,
		MetaTitle:       p.MetaTitle,
		MetaDescription: p.MetaDescription,
		CategoryName:    p.CategoryName,
		BrandName:       p.BrandName,
		PrimaryImageURL: p.PrimaryImageURL,
		Images:          imageResponses,
		CreatedAt:       p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toProductImageResponse(img models.ProductImage) dto.ProductImageResponse {
	return dto.ProductImageResponse{
		ID:        img.ID,
		ProductID: img.ProductID,
		URL:       img.URL,
		AltText:   img.AltText,
		SortOrder: img.SortOrder,
		IsPrimary: img.IsPrimary,
		CreatedAt: img.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
