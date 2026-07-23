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
	ErrProductNotFound   = errors.New("product not found")
	ErrProductSlugExists = errors.New("product slug already exists")
	ErrProductSKUExists  = errors.New("product SKU already exists")
)

type ProductRepository interface {
	Create(ctx context.Context, p models.Product) (models.Product, error)
	GetByID(ctx context.Context, id string) (models.Product, error)
	GetBySlug(ctx context.Context, slug string) (models.Product, error)
	List(ctx context.Context, query ProductListQuery) ([]models.Product, int, error)
	ListPublic(ctx context.Context, query ProductListQuery) ([]models.Product, int, error)
	Update(ctx context.Context, id string, p models.Product) (models.Product, error)
	SoftDelete(ctx context.Context, id string) error
	UpdateStock(ctx context.Context, id string, quantity int) error
}

type ProductListQuery struct {
	Search     string
	CategoryID string
	BrandID    string
	Status     string
	IsFeatured *bool
	MinPrice   float64
	MaxPrice   float64
	Sort       string
	Offset     int
	Limit      int
}

type postgresProductRepository struct {
	pool *pgxpool.Pool
}

func NewProductRepository(pool *pgxpool.Pool) ProductRepository {
	return &postgresProductRepository{pool: pool}
}

const productSelectCols = `
	p.id, COALESCE(p.name, ''), COALESCE(p.slug, ''), COALESCE(p.description, ''),
	COALESCE(p.sku, ''), p.price, p.discount_price, COALESCE(p.stock_quantity, 0),
	p.category_id, p.brand_id, COALESCE(p.status, 'active'), COALESCE(p.is_featured, false),
	p.weight, COALESCE(p.meta_title, ''), COALESCE(p.meta_description, ''),
	COALESCE(c.name, '') AS category_name,
	COALESCE(b.name, '') AS brand_name,
	COALESCE(pi.url, '') AS primary_image_url,
	p.deleted_at, p.created_at, p.updated_at
`

func scanProduct(row pgx.Row) (models.Product, error) {
	var p models.Product
	err := row.Scan(
		&p.ID, &p.Name, &p.Slug, &p.Description,
		&p.SKU, &p.Price, &p.DiscountPrice, &p.StockQuantity,
		&p.CategoryID, &p.BrandID, &p.Status, &p.IsFeatured,
		&p.Weight, &p.MetaTitle, &p.MetaDescription,
		&p.CategoryName, &p.BrandName, &p.PrimaryImageURL,
		&p.DeletedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	return p, err
}

func scanProductRows(rows pgx.Rows) (models.Product, error) {
	var p models.Product
	err := rows.Scan(
		&p.ID, &p.Name, &p.Slug, &p.Description,
		&p.SKU, &p.Price, &p.DiscountPrice, &p.StockQuantity,
		&p.CategoryID, &p.BrandID, &p.Status, &p.IsFeatured,
		&p.Weight, &p.MetaTitle, &p.MetaDescription,
		&p.CategoryName, &p.BrandName, &p.PrimaryImageURL,
		&p.DeletedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	return p, err
}

func (r *postgresProductRepository) Create(ctx context.Context, p models.Product) (models.Product, error) {
	var created models.Product
	err := r.pool.QueryRow(ctx,
		`INSERT INTO products (name, slug, description, sku, price, discount_price, stock_quantity, category_id, brand_id, status, is_featured, weight, meta_title, meta_description)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		 RETURNING `+productSelectCols,
		p.Name, p.Slug, p.Description, p.SKU, p.Price, p.DiscountPrice,
		p.StockQuantity, p.CategoryID, p.BrandID, p.Status, p.IsFeatured,
		p.Weight, p.MetaTitle, p.MetaDescription,
	).Scan(
		&created.ID, &created.Name, &created.Slug, &created.Description,
		&created.SKU, &created.Price, &created.DiscountPrice, &created.StockQuantity,
		&created.CategoryID, &created.BrandID, &created.Status, &created.IsFeatured,
		&created.Weight, &created.MetaTitle, &created.MetaDescription,
		&created.CategoryName, &created.BrandName, &created.PrimaryImageURL,
		&created.DeletedAt, &created.CreatedAt, &created.UpdatedAt,
	)
	if err != nil {
		if isDuplicateKeyError(err) {
			errStr := err.Error()
			if strings.Contains(errStr, "products_slug_key") || strings.Contains(errStr, "products_slug") {
				return models.Product{}, ErrProductSlugExists
			}
			if strings.Contains(errStr, "products_sku_key") || strings.Contains(errStr, "products_sku") {
				return models.Product{}, ErrProductSKUExists
			}
		}
		return models.Product{}, fmt.Errorf("failed to create product: %w", err)
	}
	return created, nil
}

func (r *postgresProductRepository) GetByID(ctx context.Context, id string) (models.Product, error) {
	var p models.Product
	err := r.pool.QueryRow(ctx,
		`SELECT `+productSelectCols+`
		 FROM products p
		 LEFT JOIN categories c ON c.id = p.category_id AND c.deleted_at IS NULL
		 LEFT JOIN brands b ON b.id = p.brand_id AND b.deleted_at IS NULL
		 LEFT JOIN product_images pi ON pi.product_id = p.id AND pi.is_primary = TRUE
		 WHERE p.id = $1 AND p.deleted_at IS NULL`,
		id,
	).Scan(
		&p.ID, &p.Name, &p.Slug, &p.Description,
		&p.SKU, &p.Price, &p.DiscountPrice, &p.StockQuantity,
		&p.CategoryID, &p.BrandID, &p.Status, &p.IsFeatured,
		&p.Weight, &p.MetaTitle, &p.MetaDescription,
		&p.CategoryName, &p.BrandName, &p.PrimaryImageURL,
		&p.DeletedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Product{}, ErrProductNotFound
		}
		return models.Product{}, err
	}
	return p, nil
}

func (r *postgresProductRepository) GetBySlug(ctx context.Context, slug string) (models.Product, error) {
	var p models.Product
	err := r.pool.QueryRow(ctx,
		`SELECT `+productSelectCols+`
		 FROM products p
		 LEFT JOIN categories c ON c.id = p.category_id AND c.deleted_at IS NULL
		 LEFT JOIN brands b ON b.id = p.brand_id AND b.deleted_at IS NULL
		 LEFT JOIN product_images pi ON pi.product_id = p.id AND pi.is_primary = TRUE
		 WHERE p.slug = $1 AND p.deleted_at IS NULL`,
		slug,
	).Scan(
		&p.ID, &p.Name, &p.Slug, &p.Description,
		&p.SKU, &p.Price, &p.DiscountPrice, &p.StockQuantity,
		&p.CategoryID, &p.BrandID, &p.Status, &p.IsFeatured,
		&p.Weight, &p.MetaTitle, &p.MetaDescription,
		&p.CategoryName, &p.BrandName, &p.PrimaryImageURL,
		&p.DeletedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Product{}, ErrProductNotFound
		}
		return models.Product{}, err
	}
	return p, nil
}

func (r *postgresProductRepository) buildListQuery(ctx context.Context, query ProductListQuery, includeInactive bool) ([]models.Product, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "p.deleted_at IS NULL")

	if !includeInactive {
		conditions = append(conditions, "p.status = 'active'")
	} else if query.Status != "" {
		conditions = append(conditions, fmt.Sprintf("p.status = $%d", argIdx))
		args = append(args, query.Status)
		argIdx++
	}

	if query.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(LOWER(p.name) LIKE $%d OR LOWER(p.description) LIKE $%d OR LOWER(p.sku) LIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+strings.ToLower(query.Search)+"%")
		argIdx++
	}

	if query.CategoryID != "" {
		conditions = append(conditions, fmt.Sprintf("p.category_id = $%d", argIdx))
		args = append(args, query.CategoryID)
		argIdx++
	}

	if query.BrandID != "" {
		conditions = append(conditions, fmt.Sprintf("p.brand_id = $%d", argIdx))
		args = append(args, query.BrandID)
		argIdx++
	}

	if query.IsFeatured != nil {
		conditions = append(conditions, fmt.Sprintf("p.is_featured = $%d", argIdx))
		args = append(args, *query.IsFeatured)
		argIdx++
	}

	if query.MinPrice > 0 {
		conditions = append(conditions, fmt.Sprintf("p.price >= $%d", argIdx))
		args = append(args, query.MinPrice)
		argIdx++
	}

	if query.MaxPrice > 0 {
		conditions = append(conditions, fmt.Sprintf("p.price <= $%d", argIdx))
		args = append(args, query.MaxPrice)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM products p %s", where)
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count products: %w", err)
	}

	orderBy := mapProductSortToSQL(query.Sort)

	dataQuery := fmt.Sprintf(
		`SELECT %s
		 FROM products p
		 LEFT JOIN categories c ON c.id = p.category_id AND c.deleted_at IS NULL
		 LEFT JOIN brands b ON b.id = p.brand_id AND b.deleted_at IS NULL
		 LEFT JOIN product_images pi ON pi.product_id = p.id AND pi.is_primary = TRUE
		 %s ORDER BY %s LIMIT $%d OFFSET $%d`,
		productSelectCols, where, orderBy, argIdx, argIdx+1,
	)
	args = append(args, query.Limit, query.Offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list products: %w", err)
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		p, err := scanProductRows(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, p)
	}

	return products, total, nil
}

func (r *postgresProductRepository) List(ctx context.Context, query ProductListQuery) ([]models.Product, int, error) {
	return r.buildListQuery(ctx, query, true)
}

func (r *postgresProductRepository) ListPublic(ctx context.Context, query ProductListQuery) ([]models.Product, int, error) {
	return r.buildListQuery(ctx, query, false)
}

func (r *postgresProductRepository) Update(ctx context.Context, id string, p models.Product) (models.Product, error) {
	var brandID interface{}
	if p.BrandID != nil {
		brandID = *p.BrandID
	}

	var updated models.Product
	err := r.pool.QueryRow(ctx,
		`UPDATE products SET
			name = COALESCE(NULLIF($1, ''), name),
			description = $2,
			sku = COALESCE(NULLIF($3, ''), sku),
			price = COALESCE($4, price),
			discount_price = $5,
			stock_quantity = COALESCE($6, stock_quantity),
			category_id = COALESCE(NULLIF($7, ''), category_id),
			brand_id = $8,
			status = COALESCE(NULLIF($9, ''), status),
			is_featured = COALESCE($10, is_featured),
			weight = $11,
			meta_title = $12,
			meta_description = $13,
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = $14 AND deleted_at IS NULL
		 RETURNING `+productSelectCols,
		p.Name, p.Description, p.SKU, p.Price, p.DiscountPrice,
		p.StockQuantity, p.CategoryID, brandID, p.Status, p.IsFeatured,
		p.Weight, p.MetaTitle, p.MetaDescription, id,
	).Scan(
		&updated.ID, &updated.Name, &updated.Slug, &updated.Description,
		&updated.SKU, &updated.Price, &updated.DiscountPrice, &updated.StockQuantity,
		&updated.CategoryID, &updated.BrandID, &updated.Status, &updated.IsFeatured,
		&updated.Weight, &updated.MetaTitle, &updated.MetaDescription,
		&updated.CategoryName, &updated.BrandName, &updated.PrimaryImageURL,
		&updated.DeletedAt, &updated.CreatedAt, &updated.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Product{}, ErrProductNotFound
		}
		if isDuplicateKeyError(err) {
			errStr := err.Error()
			if strings.Contains(errStr, "products_slug_key") || strings.Contains(errStr, "products_slug") {
				return models.Product{}, ErrProductSlugExists
			}
			if strings.Contains(errStr, "products_sku_key") || strings.Contains(errStr, "products_sku") {
				return models.Product{}, ErrProductSKUExists
			}
		}
		return models.Product{}, fmt.Errorf("failed to update product: %w", err)
	}
	return updated, nil
}

func (r *postgresProductRepository) SoftDelete(ctx context.Context, id string) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE products SET deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $1 AND deleted_at IS NULL`, id,
	)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrProductNotFound
	}
	return nil
}

func (r *postgresProductRepository) UpdateStock(ctx context.Context, id string, quantity int) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE products SET stock_quantity = stock_quantity + $1, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $2 AND deleted_at IS NULL`, quantity, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update stock: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrProductNotFound
	}
	return nil
}

func mapProductSortToSQL(sort string) string {
	switch sort {
	case "name":
		return "p.name ASC"
	case "-name":
		return "p.name DESC"
	case "price":
		return "p.price ASC"
	case "-price":
		return "p.price DESC"
	case "created_at":
		return "p.created_at ASC"
	case "-created_at", "":
		return "p.created_at DESC"
	default:
		return "p.created_at DESC"
	}
}
