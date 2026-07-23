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
	ErrCategoryNotFound    = errors.New("category not found")
	ErrCategoryHasProducts = errors.New("cannot delete category with products")
	ErrCategoryHasChildren = errors.New("cannot delete category with subcategories")
	ErrInvalidParent       = errors.New("invalid parent category")
	ErrCircularReference   = errors.New("category cannot be its own parent")
)

type CategoryService interface {
	Create(ctx context.Context, req dto.CreateCategoryRequest) (dto.CategoryResponse, error)
	GetByID(ctx context.Context, id string) (dto.CategoryResponse, error)
	List(ctx context.Context, search string, page, limit int) (dto.CategoryListResponse, error)
	ListActive(ctx context.Context, page, limit int) (dto.CategoryListResponse, error)
	Update(ctx context.Context, id string, req dto.UpdateCategoryRequest) (dto.CategoryResponse, error)
	Delete(ctx context.Context, id string) error
}

type categoryService struct {
	categoryRepo repositories.CategoryRepository
	productRepo  repositories.ProductRepository
}

func NewCategoryService(
	categoryRepo repositories.CategoryRepository,
	productRepo repositories.ProductRepository,
) CategoryService {
	return &categoryService{
		categoryRepo: categoryRepo,
		productRepo:  productRepo,
	}
}

func (s *categoryService) Create(ctx context.Context, req dto.CreateCategoryRequest) (dto.CategoryResponse, error) {
	if req.ParentID != "" {
		_, err := s.categoryRepo.GetByID(ctx, req.ParentID)
		if err != nil {
			return dto.CategoryResponse{}, ErrInvalidParent
		}
	}

	slug := utils.GenerateSlug(req.Name)

	var sortOrder int
	if req.ParentID != "" {
		sortOrder = 0
	}

	cat, err := s.categoryRepo.Create(ctx, req.Name, req.Description, slug, req.ParentID, sortOrder)
	if err != nil {
		if errors.Is(err, repositories.ErrCategorySlugExists) {
			slug = slug + "-" + utils.GenerateShortID()
			cat, err = s.categoryRepo.Create(ctx, req.Name, req.Description, slug, req.ParentID, sortOrder)
		}
		if err != nil {
			return dto.CategoryResponse{}, fmt.Errorf("failed to create category: %w", err)
		}
	}

	slog.Info("Category created", slog.String("category_id", cat.ID), slog.String("name", cat.Name))

	return dto.CategoryResponse{
		ID:          cat.ID,
		Name:        cat.Name,
		Description: cat.Description,
		Slug:        cat.Slug,
		ParentID:    cat.ParentID,
		IsActive:    cat.IsActive,
		SortOrder:   cat.SortOrder,
		CreatedAt:   cat.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   cat.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

func (s *categoryService) GetByID(ctx context.Context, id string) (dto.CategoryResponse, error) {
	cat, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrCategoryNotFound) {
			return dto.CategoryResponse{}, ErrCategoryNotFound
		}
		return dto.CategoryResponse{}, err
	}

	productsCount, _ := s.categoryRepo.CountProducts(ctx, id)

	return dto.CategoryResponse{
		ID:            cat.ID,
		Name:          cat.Name,
		Description:   cat.Description,
		Slug:          cat.Slug,
		ParentID:      cat.ParentID,
		IsActive:      cat.IsActive,
		SortOrder:     cat.SortOrder,
		ProductsCount: productsCount,
		CreatedAt:     cat.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:     cat.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

func (s *categoryService) List(ctx context.Context, search string, page, limit int) (dto.CategoryListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit

	categories, totalCount, err := s.categoryRepo.List(ctx, search, offset, limit)
	if err != nil {
		return dto.CategoryListResponse{}, err
	}

	responses := make([]dto.CategoryResponse, len(categories))
	for i, cat := range categories {
		productsCount, _ := s.categoryRepo.CountProducts(ctx, cat.ID)
		responses[i] = dto.CategoryResponse{
			ID:            cat.ID,
			Name:          cat.Name,
			Description:   cat.Description,
			Slug:          cat.Slug,
			ParentID:      cat.ParentID,
			IsActive:      cat.IsActive,
			SortOrder:     cat.SortOrder,
			ProductsCount: productsCount,
			CreatedAt:     cat.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:     cat.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return dto.CategoryListResponse{
		Categories: responses,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		TotalPages: repositories.CalculateTotalPages(totalCount, limit),
	}, nil
}

func (s *categoryService) ListActive(ctx context.Context, page, limit int) (dto.CategoryListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 100
	}
	offset := (page - 1) * limit

	categories, totalCount, err := s.categoryRepo.ListActive(ctx, offset, limit)
	if err != nil {
		return dto.CategoryListResponse{}, err
	}

	responses := make([]dto.CategoryResponse, len(categories))
	for i, cat := range categories {
		productsCount, _ := s.categoryRepo.CountProducts(ctx, cat.ID)
		responses[i] = dto.CategoryResponse{
			ID:            cat.ID,
			Name:          cat.Name,
			Description:   cat.Description,
			Slug:          cat.Slug,
			ParentID:      cat.ParentID,
			IsActive:      cat.IsActive,
			SortOrder:     cat.SortOrder,
			ProductsCount: productsCount,
			CreatedAt:     cat.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:     cat.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return dto.CategoryListResponse{
		Categories: responses,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		TotalPages: repositories.CalculateTotalPages(totalCount, limit),
	}, nil
}

func (s *categoryService) Update(ctx context.Context, id string, req dto.UpdateCategoryRequest) (dto.CategoryResponse, error) {
	if req.ParentID != "" && req.ParentID == id {
		return dto.CategoryResponse{}, ErrCircularReference
	}

	if req.ParentID != "" {
		_, err := s.categoryRepo.GetByID(ctx, req.ParentID)
		if err != nil {
			return dto.CategoryResponse{}, ErrInvalidParent
		}
	}

	cat, err := s.categoryRepo.Update(ctx, id, req.Name, req.Description, req.ParentID, req.IsActive, req.SortOrder)
	if err != nil {
		if errors.Is(err, repositories.ErrCategoryNotFound) {
			return dto.CategoryResponse{}, ErrCategoryNotFound
		}
		return dto.CategoryResponse{}, err
	}

	productsCount, _ := s.categoryRepo.CountProducts(ctx, id)

	return dto.CategoryResponse{
		ID:            cat.ID,
		Name:          cat.Name,
		Description:   cat.Description,
		Slug:          cat.Slug,
		ParentID:      cat.ParentID,
		IsActive:      cat.IsActive,
		SortOrder:     cat.SortOrder,
		ProductsCount: productsCount,
		CreatedAt:     cat.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:     cat.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

func (s *categoryService) Delete(ctx context.Context, id string) error {
	productsCount, err := s.categoryRepo.CountProducts(ctx, id)
	if err != nil {
		return err
	}
	if productsCount > 0 {
		return ErrCategoryHasProducts
	}

	childrenCount, err := s.categoryRepo.CountChildren(ctx, id)
	if err != nil {
		return err
	}
	if childrenCount > 0 {
		err = s.categoryRepo.ReassignChildren(ctx, id, "")
		if err != nil {
			return fmt.Errorf("failed to reassign children: %w", err)
		}
	}

	err = s.categoryRepo.SoftDelete(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrCategoryNotFound) {
			return ErrCategoryNotFound
		}
		return err
	}

	slog.Info("Category deleted", slog.String("category_id", id))
	return nil
}


