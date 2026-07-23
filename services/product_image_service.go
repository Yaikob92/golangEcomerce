package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"backend/dto"
	"backend/models"
	"backend/repositories"
)

var (
	ErrImageNotFound     = errors.New("image not found")
	ErrImageLimitReached = errors.New("image limit reached (max 10)")
)

type ProductImageService interface {
	GetImages(ctx context.Context, productID string) ([]dto.ProductImageResponse, error)
	AddImage(ctx context.Context, productID string, req dto.AddProductImageRequest) (dto.ProductImageResponse, error)
	RemoveImage(ctx context.Context, imageID string) error
	SetPrimary(ctx context.Context, productID, imageID string) error
	ReorderImages(ctx context.Context, productID string, req dto.UpdateImageOrderRequest) error
}

type productImageService struct {
	imageRepo  repositories.ProductImageRepository
	productRepo repositories.ProductRepository
}

func NewProductImageService(
	imageRepo repositories.ProductImageRepository,
	productRepo repositories.ProductRepository,
) ProductImageService {
	return &productImageService{
		imageRepo:   imageRepo,
		productRepo: productRepo,
	}
}

func (s *productImageService) GetImages(ctx context.Context, productID string) ([]dto.ProductImageResponse, error) {
	images, err := s.imageRepo.GetByProductID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get images: %w", err)
	}

	responses := make([]dto.ProductImageResponse, len(images))
	for i, img := range images {
		responses[i] = toImageResponse(img)
	}
	return responses, nil
}

func (s *productImageService) AddImage(ctx context.Context, productID string, req dto.AddProductImageRequest) (dto.ProductImageResponse, error) {
	_, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, repositories.ErrProductNotFound) {
			return dto.ProductImageResponse{}, ErrProductNotFound
		}
		return dto.ProductImageResponse{}, fmt.Errorf("failed to get product: %w", err)
	}

	img, err := s.imageRepo.Create(ctx, productID, req.URL, req.AltText, req.IsPrimary)
	if err != nil {
		if errors.Is(err, repositories.ErrImageLimitReached) {
			return dto.ProductImageResponse{}, ErrImageLimitReached
		}
		return dto.ProductImageResponse{}, fmt.Errorf("failed to add image: %w", err)
	}

	slog.Info("Product image added", slog.String("product_id", productID), slog.String("image_id", img.ID))
	return toImageResponse(img), nil
}

func (s *productImageService) RemoveImage(ctx context.Context, imageID string) error {
	err := s.imageRepo.Delete(ctx, imageID)
	if err != nil {
		if errors.Is(err, repositories.ErrImageNotFound) {
			return ErrImageNotFound
		}
		return fmt.Errorf("failed to remove image: %w", err)
	}

	slog.Info("Product image removed", slog.String("image_id", imageID))
	return nil
}

func (s *productImageService) SetPrimary(ctx context.Context, productID, imageID string) error {
	err := s.imageRepo.SetPrimary(ctx, productID, imageID)
	if err != nil {
		if errors.Is(err, repositories.ErrImageNotFound) {
			return ErrImageNotFound
		}
		return fmt.Errorf("failed to set primary image: %w", err)
	}

	slog.Info("Primary image updated", slog.String("product_id", productID), slog.String("image_id", imageID))
	return nil
}

func (s *productImageService) ReorderImages(ctx context.Context, productID string, req dto.UpdateImageOrderRequest) error {
	err := s.imageRepo.UpdateSortOrder(ctx, req.ImageIDs)
	if err != nil {
		return fmt.Errorf("failed to reorder images: %w", err)
	}

	slog.Info("Product images reordered", slog.String("product_id", productID))
	return nil
}

func toImageResponse(img models.ProductImage) dto.ProductImageResponse {
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
