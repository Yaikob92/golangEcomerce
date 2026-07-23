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
	ErrCategoryNotFound    = errors.New("category not found")
	ErrCategorySlugExists  = errors.New("category slug already exists")
	ErrCategoryHasProducts = errors.New("category has products")
	ErrCategoryHasChildren = errors.New("category has subcategories")
)

type CategoryRepository interface {
	Create(ctx context.Context, name, description, slug, parentID string, sortOrder int) (models.Category, error)
	GetByID(ctx context.Context, id string) (models.Category, error)
	GetBySlug(ctx context.Context, slug string) (models.Category, error)
	List(ctx context.Context, search string, offset, limit int) ([]models.Category, int, error)
	ListActive(ctx context.Context, offset, limit int) ([]models.Category, int, error)
	Update(ctx context.Context, id, name, description, parentID string, isActive *bool, sortOrder *int) (models.Category, error)
	SoftDelete(ctx context.Context, id string) error
	CountProducts(ctx context.Context, categoryID string) (int, error)
	CountChildren(ctx context.Context, parentID string) (int, error)
	ReassignChildren(ctx context.Context, oldParentID string, newParentID string) error
}

type postgresCategoryRepository struct {
	pool *pgxpool.Pool
}

func NewCategoryRepository(pool *pgxpool.Pool) CategoryRepository {
	return &postgresCategoryRepository{pool: pool}
}

const categorySelectCols = `
	id, COALESCE(name, ''), COALESCE(description, ''), slug,
	parent_id, COALESCE(is_active, true), COALESCE(sort_order, 0),
	deleted_at, created_at, updated_at
`

func scanCategory(row pgx.Row) (models.Category, error) {
	var c models.Category
	err := row.Scan(
		&c.ID, &c.Name, &c.Description, &c.Slug,
		&c.ParentID, &c.IsActive, &c.SortOrder,
		&c.DeletedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	return c, err
}

func scanCategoryRows(rows pgx.Rows) (models.Category, error) {
	var c models.Category
	err := rows.Scan(
		&c.ID, &c.Name, &c.Description, &c.Slug,
		&c.ParentID, &c.IsActive, &c.SortOrder,
		&c.DeletedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	return c, err
}

func (r *postgresCategoryRepository) Create(ctx context.Context, name, description, slug, parentID string, sortOrder int) (models.Category, error) {
	var parentIDPtr *string
	if parentID != "" {
		parentIDPtr = &parentID
	}

	var c models.Category
	err := r.pool.QueryRow(ctx,
		`INSERT INTO categories (name, description, slug, parent_id, sort_order)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+categorySelectCols,
		name, description, slug, parentIDPtr, sortOrder,
	).Scan(
		&c.ID, &c.Name, &c.Description, &c.Slug,
		&c.ParentID, &c.IsActive, &c.SortOrder,
		&c.DeletedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if isDuplicateKeyError(err) {
			return models.Category{}, ErrCategorySlugExists
		}
		return models.Category{}, fmt.Errorf("failed to create category: %w", err)
	}
	return c, nil
}

func (r *postgresCategoryRepository) GetByID(ctx context.Context, id string) (models.Category, error) {
	var c models.Category
	err := r.pool.QueryRow(ctx,
		`SELECT `+categorySelectCols+`
		 FROM categories WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(
		&c.ID, &c.Name, &c.Description, &c.Slug,
		&c.ParentID, &c.IsActive, &c.SortOrder,
		&c.DeletedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Category{}, ErrCategoryNotFound
		}
		return models.Category{}, err
	}
	return c, nil
}

func (r *postgresCategoryRepository) GetBySlug(ctx context.Context, slug string) (models.Category, error) {
	var c models.Category
	err := r.pool.QueryRow(ctx,
		`SELECT `+categorySelectCols+`
		 FROM categories WHERE slug = $1 AND deleted_at IS NULL`,
		slug,
	).Scan(
		&c.ID, &c.Name, &c.Description, &c.Slug,
		&c.ParentID, &c.IsActive, &c.SortOrder,
		&c.DeletedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Category{}, ErrCategoryNotFound
		}
		return models.Category{}, err
	}
	return c, nil
}

func (r *postgresCategoryRepository) List(ctx context.Context, search string, offset, limit int) ([]models.Category, int, error) {
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
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM categories %s", where)
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count categories: %w", err)
	}

	dataQuery := fmt.Sprintf(
		`SELECT %s FROM categories %s ORDER BY sort_order ASC, name ASC LIMIT $%d OFFSET $%d`,
		categorySelectCols, where, argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list categories: %w", err)
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		c, err := scanCategoryRows(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, c)
	}

	return categories, total, nil
}

func (r *postgresCategoryRepository) ListActive(ctx context.Context, offset, limit int) ([]models.Category, int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM categories WHERE deleted_at IS NULL AND is_active = TRUE",
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count active categories: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT `+categorySelectCols+`
		 FROM categories WHERE deleted_at IS NULL AND is_active = TRUE
		 ORDER BY sort_order ASC, name ASC
		 LIMIT $1 OFFSET $2`, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list active categories: %w", err)
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		c, err := scanCategoryRows(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, c)
	}

	return categories, total, nil
}

func (r *postgresCategoryRepository) Update(ctx context.Context, id, name, description, parentID string, isActive *bool, sortOrder *int) (models.Category, error) {
	var parentIDPtr *string
	if parentID != "" {
		parentIDPtr = &parentID
	}

	nameExpr := "name"
	nameArg := name
	if name == "" {
		nameExpr = "name"
		nameArg = ""
	}

	var c models.Category
	err := r.pool.QueryRow(ctx,
		`UPDATE categories SET
			name = COALESCE(NULLIF($1, ''), name),
			description = $2,
			parent_id = $3,
			is_active = COALESCE($4, is_active),
			sort_order = COALESCE($5, sort_order),
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = $6 AND deleted_at IS NULL
		 RETURNING `+categorySelectCols,
		name, description, parentIDPtr, isActive, sortOrder, id,
	).Scan(
		&c.ID, &c.Name, &c.Description, &c.Slug,
		&c.ParentID, &c.IsActive, &c.SortOrder,
		&c.DeletedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	_ = nameExpr
	_ = nameArg
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Category{}, ErrCategoryNotFound
		}
		if isDuplicateKeyError(err) {
			return models.Category{}, ErrCategorySlugExists
		}
		return models.Category{}, fmt.Errorf("failed to update category: %w", err)
	}
	return c, nil
}

func (r *postgresCategoryRepository) SoftDelete(ctx context.Context, id string) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE categories SET deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $1 AND deleted_at IS NULL`, id,
	)
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrCategoryNotFound
	}
	return nil
}

func (r *postgresCategoryRepository) CountProducts(ctx context.Context, categoryID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM products WHERE category_id = $1 AND deleted_at IS NULL",
		categoryID,
	).Scan(&count)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "relation") {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

func (r *postgresCategoryRepository) CountChildren(ctx context.Context, parentID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM categories WHERE parent_id = $1 AND deleted_at IS NULL",
		parentID,
	).Scan(&count)
	return count, err
}

func (r *postgresCategoryRepository) ReassignChildren(ctx context.Context, oldParentID string, newParentID string) error {
	var newParentIDPtr *string
	if newParentID != "" {
		newParentIDPtr = &newParentID
	}
	_, err := r.pool.Exec(ctx,
		"UPDATE categories SET parent_id = $1, updated_at = CURRENT_TIMESTAMP WHERE parent_id = $2 AND deleted_at IS NULL",
		newParentIDPtr, oldParentID,
	)
	return err
}
