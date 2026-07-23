package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"backend/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrBrandNotFound   = errors.New("brand not found")
	ErrBrandSlugExists = errors.New("brand slug already exists")
	ErrBrandHasProducts = errors.New("brand has products")
)

type BrandRepository interface {
	Create(ctx context.Context, name, description, slug, logoURL string) (models.Brand, error)
	GetByID(ctx context.Context, id string) (models.Brand, error)
	GetBySlug(ctx context.Context, slug string) (models.Brand, error)
	List(ctx context.Context, search string, offset, limit int) ([]models.Brand, int, error)
	ListActive(ctx context.Context, offset, limit int) ([]models.Brand, int, error)
	Update(ctx context.Context, id, name, description, logoURL string, isActive *bool) (models.Brand, error)
	SoftDelete(ctx context.Context, id string) error
	CountProducts(ctx context.Context, brandID string) (int, error)
}

type postgresBrandRepository struct {
	pool *pgxpool.Pool
}

func NewBrandRepository(pool *pgxpool.Pool) BrandRepository {
	return &postgresBrandRepository{pool: pool}
}

const brandSelectCols = `
	id, COALESCE(name, ''), slug, COALESCE(description, ''),
	COALESCE(logo_url, ''), COALESCE(is_active, true),
	deleted_at, created_at, updated_at
`

func scanBrand(row pgx.Row) (models.Brand, error) {
	var b models.Brand
	err := row.Scan(
		&b.ID, &b.Name, &b.Slug, &b.Description,
		&b.LogoURL, &b.IsActive,
		&b.DeletedAt, &b.CreatedAt, &b.UpdatedAt,
	)
	return b, err
}

func scanBrandRows(rows pgx.Rows) (models.Brand, error) {
	var b models.Brand
	err := rows.Scan(
		&b.ID, &b.Name, &b.Slug, &b.Description,
		&b.LogoURL, &b.IsActive,
		&b.DeletedAt, &b.CreatedAt, &b.UpdatedAt,
	)
	return b, err
}

func (r *postgresBrandRepository) Create(ctx context.Context, name, description, slug, logoURL string) (models.Brand, error) {
	var b models.Brand
	err := r.pool.QueryRow(ctx,
		`INSERT INTO brands (name, description, slug, logo_url)
		 VALUES ($1, $2, $3, $4)
		 RETURNING `+brandSelectCols,
		name, description, slug, logoURL,
	).Scan(
		&b.ID, &b.Name, &b.Slug, &b.Description,
		&b.LogoURL, &b.IsActive,
		&b.DeletedAt, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if isDuplicateKeyError(err) {
			return models.Brand{}, ErrBrandSlugExists
		}
		return models.Brand{}, fmt.Errorf("failed to create brand: %w", err)
	}
	return b, nil
}

func (r *postgresBrandRepository) GetByID(ctx context.Context, id string) (models.Brand, error) {
	var b models.Brand
	err := r.pool.QueryRow(ctx,
		`SELECT `+brandSelectCols+`
		 FROM brands WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(
		&b.ID, &b.Name, &b.Slug, &b.Description,
		&b.LogoURL, &b.IsActive,
		&b.DeletedAt, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Brand{}, ErrBrandNotFound
		}
		return models.Brand{}, err
	}
	return b, nil
}

func (r *postgresBrandRepository) GetBySlug(ctx context.Context, slug string) (models.Brand, error) {
	var b models.Brand
	err := r.pool.QueryRow(ctx,
		`SELECT `+brandSelectCols+`
		 FROM brands WHERE slug = $1 AND deleted_at IS NULL`,
		slug,
	).Scan(
		&b.ID, &b.Name, &b.Slug, &b.Description,
		&b.LogoURL, &b.IsActive,
		&b.DeletedAt, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Brand{}, ErrBrandNotFound
		}
		return models.Brand{}, err
	}
	return b, nil
}

func (r *postgresBrandRepository) List(ctx context.Context, search string, offset, limit int) ([]models.Brand, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "deleted_at IS NULL")

	if search != "" {
		conditions = append(conditions, fmt.Sprintf("(LOWER(name) LIKE $%d)", argIdx))
		args = append(args, "%"+strings.ToLower(search)+"%")
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM brands %s", where)
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count brands: %w", err)
	}

	dataQuery := fmt.Sprintf(
		`SELECT %s FROM brands %s ORDER BY name ASC LIMIT $%d OFFSET $%d`,
		brandSelectCols, where, argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list brands: %w", err)
	}
	defer rows.Close()

	var brands []models.Brand
	for rows.Next() {
		b, err := scanBrandRows(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan brand: %w", err)
		}
		brands = append(brands, b)
	}

	return brands, total, nil
}

func (r *postgresBrandRepository) ListActive(ctx context.Context, offset, limit int) ([]models.Brand, int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM brands WHERE deleted_at IS NULL AND is_active = TRUE",
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count active brands: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT `+brandSelectCols+`
		 FROM brands WHERE deleted_at IS NULL AND is_active = TRUE
		 ORDER BY name ASC
		 LIMIT $1 OFFSET $2`, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list active brands: %w", err)
	}
	defer rows.Close()

	var brands []models.Brand
	for rows.Next() {
		b, err := scanBrandRows(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan brand: %w", err)
		}
		brands = append(brands, b)
	}

	return brands, total, nil
}

func (r *postgresBrandRepository) Update(ctx context.Context, id, name, description, logoURL string, isActive *bool) (models.Brand, error) {
	var b models.Brand
	err := r.pool.QueryRow(ctx,
		`UPDATE brands SET
			name = COALESCE(NULLIF($1, ''), name),
			description = $2,
			logo_url = $3,
			is_active = COALESCE($4, is_active),
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = $5 AND deleted_at IS NULL
		 RETURNING `+brandSelectCols,
		name, description, logoURL, isActive, id,
	).Scan(
		&b.ID, &b.Name, &b.Slug, &b.Description,
		&b.LogoURL, &b.IsActive,
		&b.DeletedAt, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Brand{}, ErrBrandNotFound
		}
		if isDuplicateKeyError(err) {
			return models.Brand{}, ErrBrandSlugExists
		}
		return models.Brand{}, fmt.Errorf("failed to update brand: %w", err)
	}
	return b, nil
}

func (r *postgresBrandRepository) SoftDelete(ctx context.Context, id string) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE brands SET deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $1 AND deleted_at IS NULL`, id,
	)
	if err != nil {
		return fmt.Errorf("failed to delete brand: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrBrandNotFound
	}
	return nil
}

func (r *postgresBrandRepository) CountProducts(ctx context.Context, brandID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM products WHERE brand_id = $1 AND deleted_at IS NULL",
		brandID,
	).Scan(&count)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "relation") {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}
