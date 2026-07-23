package models

import "time"

type Product struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Slug            string     `json:"slug"`
	Description     string     `json:"description"`
	SKU             string     `json:"sku"`
	Price           float64    `json:"price"`
	DiscountPrice   *float64   `json:"discount_price,omitempty"`
	StockQuantity   int        `json:"stock_quantity"`
	CategoryID      string     `json:"category_id"`
	BrandID         *string    `json:"brand_id,omitempty"`
	Status          string     `json:"status"`
	IsFeatured      bool       `json:"is_featured"`
	Weight          *float64   `json:"weight,omitempty"`
	MetaTitle       string     `json:"meta_title"`
	MetaDescription string     `json:"meta_description"`
	CategoryName    string     `json:"category_name,omitempty"`
	BrandName       string     `json:"brand_name,omitempty"`
	PrimaryImageURL string     `json:"primary_image_url,omitempty"`
	Images          []ProductImage `json:"images,omitempty"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
