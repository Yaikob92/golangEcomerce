package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"backend/dto"
	"backend/repositories"
	"backend/utils"
)

var (
	ErrBrandNotFound     = errors.New("brand not found")
	ErrBrandHasProducts  = errors.New("cannot delete brand with products")
)

type BrandService interface {
	Create(ctx context.Context, req dto.CreateBrandRequest) (dto.BrandResponse, error)
	GetByID(ctx context.Context, id string) (dto.BrandResponse, error)
	List(ctx context.Context, search string, page, limit int) (dto.BrandListResponse, error)
	ListActive(ctx context.Context, page, limit int) (dto.BrandListResponse, error)
	Update(ctx context.Context, id string, req dto.UpdateBrandRequest) (dto.BrandResponse, error)
	Delete(ctx context.Context, id string) error
}

type brandService struct {
	brandRepo   repositories.BrandRepository
	productRepo repositories.ProductRepository
}

func NewBrandService(
	brandRepo repositories.BrandRepository,
	productRepo repositories.ProductRepository,
) BrandService {
	return &brandService{
		brandRepo:   brandRepo,
		productRepo: productRepo,
	}
}

func (s *brandService) Create(ctx context.Context, req dto.CreateBrandRequest) (dto.BrandResponse, error) {
	slug := utils.GenerateSlug(req.Name)

	brand, err := s.brandRepo.Create(ctx, req.Name, req.Description, slug, req.LogoURL)
	if err != nil {
		if errors.Is(err, repositories.ErrBrandSlugExists) {
			slug = slug + "-" + utils.GenerateShortID()
			brand, err = s.brandRepo.Create(ctx, req.Name, req.Description, slug, req.LogoURL)
		}
		if err != nil {
			return dto.BrandResponse{}, fmt.Errorf("failed to create brand: %w", err)
		}
	}

	slog.Info("Brand created", slog.String("brand_id", brand.ID), slog.String("name", brand.Name))

	return dto.BrandResponse{
		ID:          brand.ID,
		Name:        brand.Name,
		Slug:        brand.Slug,
		Description: brand.Description,
		LogoURL:     brand.LogoURL,
		IsActive:    brand.IsActive,
		CreatedAt:   brand.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   brand.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

func (s *brandService) GetByID(ctx context.Context, id string) (dto.BrandResponse, error) {
	brand, err := s.brandRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrBrandNotFound) {
			return dto.BrandResponse{}, ErrBrandNotFound
		}
		return dto.BrandResponse{}, err
	}

	productsCount, _ := s.brandRepo.CountProducts(ctx, brand.ID)

	return dto.BrandResponse{
		ID:            brand.ID,
		Name:          brand.Name,
		Slug:          brand.Slug,
		Description:   brand.Description,
		LogoURL:       brand.LogoURL,
		IsActive:      brand.IsActive,
		ProductsCount: productsCount,
		CreatedAt:     brand.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:     brand.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

func (s *brandService) List(ctx context.Context, search string, page, limit int) (dto.BrandListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit

	brands, totalCount, err := s.brandRepo.List(ctx, search, offset, limit)
	if err != nil {
		return dto.BrandListResponse{}, err
	}

	responses := make([]dto.BrandResponse, len(brands))
	for i, brand := range brands {
		productsCount, _ := s.brandRepo.CountProducts(ctx, brand.ID)
		responses[i] = dto.BrandResponse{
			ID:            brand.ID,
			Name:          brand.Name,
			Slug:          brand.Slug,
			Description:   brand.Description,
			LogoURL:       brand.LogoURL,
			IsActive:      brand.IsActive,
			ProductsCount: productsCount,
			CreatedAt:     brand.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:     brand.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return dto.BrandListResponse{
		Brands:     responses,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		TotalPages: repositories.CalculateTotalPages(totalCount, limit),
	}, nil
}

func (s *brandService) ListActive(ctx context.Context, page, limit int) (dto.BrandListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 100
	}
	offset := (page - 1) * limit

	brands, totalCount, err := s.brandRepo.ListActive(ctx, offset, limit)
	if err != nil {
		return dto.BrandListResponse{}, err
	}

	responses := make([]dto.BrandResponse, len(brands))
	for i, brand := range brands {
		productsCount, _ := s.brandRepo.CountProducts(ctx, brand.ID)
		responses[i] = dto.BrandResponse{
			ID:            brand.ID,
			Name:          brand.Name,
			Slug:          brand.Slug,
			Description:   brand.Description,
			LogoURL:       brand.LogoURL,
			IsActive:      brand.IsActive,
			ProductsCount: productsCount,
			CreatedAt:     brand.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:     brand.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return dto.BrandListResponse{
		Brands:     responses,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		TotalPages: repositories.CalculateTotalPages(totalCount, limit),
	}, nil
}

func (s *brandService) Update(ctx context.Context, id string, req dto.UpdateBrandRequest) (dto.BrandResponse, error) {
	brand, err := s.brandRepo.Update(ctx, id, req.Name, req.Description, req.LogoURL, req.IsActive)
	if err != nil {
		if errors.Is(err, repositories.ErrBrandNotFound) {
			return dto.BrandResponse{}, ErrBrandNotFound
		}
		if errors.Is(err, repositories.ErrBrandSlugExists) {
			return dto.BrandResponse{}, fmt.Errorf("brand slug already exists: %w", err)
		}
		return dto.BrandResponse{}, err
	}

	productsCount, _ := s.brandRepo.CountProducts(ctx, brand.ID)

	return dto.BrandResponse{
		ID:            brand.ID,
		Name:          brand.Name,
		Slug:          brand.Slug,
		Description:   brand.Description,
		LogoURL:       brand.LogoURL,
		IsActive:      brand.IsActive,
		ProductsCount: productsCount,
		CreatedAt:     brand.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:     brand.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

func (s *brandService) Delete(ctx context.Context, id string) error {
	productsCount, err := s.brandRepo.CountProducts(ctx, id)
	if err != nil {
		return err
	}
	if productsCount > 0 {
		return ErrBrandHasProducts
	}

	err = s.brandRepo.SoftDelete(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrBrandNotFound) {
			return ErrBrandNotFound
		}
		return err
	}

	slog.Info("Brand deleted", slog.String("brand_id", id))
	return nil
}
