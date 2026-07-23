package repositories

import (
	"context"
	"errors"
	"fmt"

	"backend/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrImageNotFound = errors.New("image not found")
	ErrImageLimitReached = errors.New("image limit reached (max 10)")
)

type ProductImageRepository interface {
	Create(ctx context.Context, productID, url, altText string, isPrimary bool) (models.ProductImage, error)
	GetByProductID(ctx context.Context, productID string) ([]models.ProductImage, error)
	Delete(ctx context.Context, id string) error
	SetPrimary(ctx context.Context, productID, imageID string) error
	UpdateSortOrder(ctx context.Context, imageIDs []string) error
	CountByProduct(ctx context.Context, productID string) (int, error)
}

type postgresProductImageRepository struct {
	pool *pgxpool.Pool
}

func NewProductImageRepository(pool *pgxpool.Pool) ProductImageRepository {
	return &postgresProductImageRepository{pool: pool}
}

const imageSelectCols = `
	id, product_id, url, COALESCE(alt_text, ''), sort_order, is_primary, created_at
`

func scanImage(row pgx.Row) (models.ProductImage, error) {
	var img models.ProductImage
	err := row.Scan(
		&img.ID, &img.ProductID, &img.URL, &img.AltText,
		&img.SortOrder, &img.IsPrimary, &img.CreatedAt,
	)
	return img, err
}

func (r *postgresProductImageRepository) Create(ctx context.Context, productID, url, altText string, isPrimary bool) (models.ProductImage, error) {
	count, err := r.CountByProduct(ctx, productID)
	if err != nil {
		return models.ProductImage{}, fmt.Errorf("failed to count images: %w", err)
	}
	if count >= 10 {
		return models.ProductImage{}, ErrImageLimitReached
	}

	if isPrimary {
		_, err = r.pool.Exec(ctx,
			"UPDATE product_images SET is_primary = FALSE WHERE product_id = $1", productID,
		)
		if err != nil {
			return models.ProductImage{}, fmt.Errorf("failed to reset primary: %w", err)
		}
	}

	var img models.ProductImage
	err = r.pool.QueryRow(ctx,
		`INSERT INTO product_images (product_id, url, alt_text, sort_order, is_primary)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+imageSelectCols,
		productID, url, altText, count, isPrimary,
	).Scan(
		&img.ID, &img.ProductID, &img.URL, &img.AltText,
		&img.SortOrder, &img.IsPrimary, &img.CreatedAt,
	)
	if err != nil {
		return models.ProductImage{}, fmt.Errorf("failed to create image: %w", err)
	}
	return img, nil
}

func (r *postgresProductImageRepository) GetByProductID(ctx context.Context, productID string) ([]models.ProductImage, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+imageSelectCols+`
		 FROM product_images WHERE product_id = $1
		 ORDER BY sort_order ASC, created_at ASC`, productID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}
	defer rows.Close()

	var images []models.ProductImage
	for rows.Next() {
		var img models.ProductImage
		err := rows.Scan(
			&img.ID, &img.ProductID, &img.URL, &img.AltText,
			&img.SortOrder, &img.IsPrimary, &img.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan image: %w", err)
		}
		images = append(images, img)
	}
	return images, nil
}

func (r *postgresProductImageRepository) Delete(ctx context.Context, id string) error {
	result, err := r.pool.Exec(ctx,
		"DELETE FROM product_images WHERE id = $1", id,
	)
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrImageNotFound
	}
	return nil
}

func (r *postgresProductImageRepository) SetPrimary(ctx context.Context, productID, imageID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		"UPDATE product_images SET is_primary = FALSE WHERE product_id = $1", productID,
	)
	if err != nil {
		return fmt.Errorf("failed to reset primary: %w", err)
	}

	result, err := tx.Exec(ctx,
		"UPDATE product_images SET is_primary = TRUE WHERE id = $1 AND product_id = $2", imageID, productID,
	)
	if err != nil {
		return fmt.Errorf("failed to set primary: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrImageNotFound
	}

	return tx.Commit(ctx)
}

func (r *postgresProductImageRepository) UpdateSortOrder(ctx context.Context, imageIDs []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for i, id := range imageIDs {
		_, err := tx.Exec(ctx,
			"UPDATE product_images SET sort_order = $1 WHERE id = $2", i, id,
		)
		if err != nil {
			return fmt.Errorf("failed to update sort order: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *postgresProductImageRepository) CountByProduct(ctx context.Context, productID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM product_images WHERE product_id = $1", productID,
	).Scan(&count)
	return count, err
}
