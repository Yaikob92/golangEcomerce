package dto

// ── Category ──

type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	ParentID    string `json:"parent_id"`
}

type UpdateCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ParentID    string `json:"parent_id"`
	IsActive    *bool  `json:"is_active"`
	SortOrder   *int   `json:"sort_order"`
}

type CategoryResponse struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	Description    string             `json:"description"`
	Slug           string             `json:"slug"`
	ParentID       *string            `json:"parent_id,omitempty"`
	IsActive       bool               `json:"is_active"`
	SortOrder      int                `json:"sort_order"`
	ProductsCount  int                `json:"products_count"`
	Children       []CategoryResponse `json:"children,omitempty"`
	CreatedAt      string             `json:"created_at"`
	UpdatedAt      string             `json:"updated_at"`
}

type CategoryListResponse struct {
	Categories  []CategoryResponse `json:"categories"`
	TotalCount  int                `json:"total_count"`
	Page        int                `json:"page"`
	Limit       int                `json:"limit"`
	TotalPages  int                `json:"total_pages"`
}

// ── Brand ──

type CreateBrandRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	LogoURL     string `json:"logo_url"`
}

type UpdateBrandRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	LogoURL     string `json:"logo_url"`
	IsActive    *bool  `json:"is_active"`
}

type BrandResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	Description   string `json:"description"`
	LogoURL       string `json:"logo_url"`
	IsActive      bool   `json:"is_active"`
	ProductsCount int    `json:"products_count"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type BrandListResponse struct {
	Brands      []BrandResponse `json:"brands"`
	TotalCount  int             `json:"total_count"`
	Page        int             `json:"page"`
	Limit       int             `json:"limit"`
	TotalPages  int             `json:"total_pages"`
}

// ── Product ──

type CreateProductRequest struct {
	Name            string   `json:"name" binding:"required"`
	Description     string   `json:"description"`
	SKU             string   `json:"sku"`
	Price           float64  `json:"price" binding:"required,gt=0"`
	DiscountPrice   *float64 `json:"discount_price"`
	StockQuantity   int      `json:"stock_quantity" binding:"min=0"`
	CategoryID      string   `json:"category_id" binding:"required"`
	BrandID         string   `json:"brand_id"`
	Status          string   `json:"status" binding:"omitempty,oneof=active draft archived"`
	IsFeatured      bool     `json:"is_featured"`
	Weight          *float64 `json:"weight"`
	MetaTitle       string   `json:"meta_title"`
	MetaDescription string   `json:"meta_description"`
}

type UpdateProductRequest struct {
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	SKU             string   `json:"sku"`
	Price           *float64 `json:"price"`
	DiscountPrice   *float64 `json:"discount_price"`
	StockQuantity   *int     `json:"stock_quantity"`
	CategoryID      string   `json:"category_id"`
	BrandID         string   `json:"brand_id"`
	Status          string   `json:"status" binding:"omitempty,oneof=active draft archived"`
	IsFeatured      *bool    `json:"is_featured"`
	Weight          *float64 `json:"weight"`
	MetaTitle       string   `json:"meta_title"`
	MetaDescription string   `json:"meta_description"`
}

type ProductListQuery struct {
	Page       int     `form:"page" binding:"omitempty,min=1"`
	Limit      int     `form:"limit" binding:"omitempty,min=1,max=100"`
	Search     string  `form:"search"`
	CategoryID string  `form:"category_id"`
	BrandID    string  `form:"brand_id"`
	Status     string  `form:"status" binding:"omitempty,oneof=active draft archived"`
	IsFeatured *bool   `form:"is_featured"`
	MinPrice   float64 `form:"min_price" binding:"omitempty,min=0"`
	MaxPrice   float64 `form:"max_price" binding:"omitempty,min=0"`
	Sort       string  `form:"sort" binding:"omitempty,oneof=name price created_at -name -price -created_at"`
}

func (q *ProductListQuery) SetDefaults() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Sort == "" {
		q.Sort = "-created_at"
	}
}

type ProductResponse struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Slug            string           `json:"slug"`
	Description     string           `json:"description"`
	SKU             string           `json:"sku"`
	Price           float64          `json:"price"`
	DiscountPrice   *float64         `json:"discount_price,omitempty"`
	StockQuantity   int              `json:"stock_quantity"`
	CategoryID      string           `json:"category_id"`
	BrandID         *string          `json:"brand_id,omitempty"`
	Status          string           `json:"status"`
	IsFeatured      bool             `json:"is_featured"`
	Weight          *float64         `json:"weight,omitempty"`
	MetaTitle       string           `json:"meta_title"`
	MetaDescription string           `json:"meta_description"`
	CategoryName    string           `json:"category_name,omitempty"`
	BrandName       string           `json:"brand_name,omitempty"`
	PrimaryImageURL string           `json:"primary_image_url,omitempty"`
	Images          []ProductImageResponse `json:"images,omitempty"`
	CreatedAt       string           `json:"created_at"`
	UpdatedAt       string           `json:"updated_at"`
}

type ProductListResponse struct {
	Products    []ProductResponse `json:"products"`
	TotalCount  int               `json:"total_count"`
	Page        int               `json:"page"`
	Limit       int               `json:"limit"`
	TotalPages  int               `json:"total_pages"`
}

// ── Product Image ──

type AddProductImageRequest struct {
	URL      string `json:"url" binding:"required"`
	AltText  string `json:"alt_text"`
	IsPrimary bool  `json:"is_primary"`
}

type UpdateImageOrderRequest struct {
	ImageIDs []string `json:"image_ids" binding:"required"`
}

type ProductImageResponse struct {
	ID        string `json:"id"`
	ProductID string `json:"product_id"`
	URL       string `json:"url"`
	AltText   string `json:"alt_text"`
	SortOrder int    `json:"sort_order"`
	IsPrimary bool   `json:"is_primary"`
	CreatedAt string `json:"created_at"`
}
