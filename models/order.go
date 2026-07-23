package models

import "time"

type Order struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	OrderNumber     string     `json:"order_number"`
	Status          string     `json:"status"`
	Subtotal        float64    `json:"subtotal"`
	ShippingCost    float64    `json:"shipping_cost"`
	Tax             float64    `json:"tax"`
	Discount        float64    `json:"discount"`
	TotalAmount     float64    `json:"total_amount"`
	Currency        string     `json:"currency"`
	ShippingAddress *string    `json:"shipping_address,omitempty"`
	BillingAddress  *string    `json:"billing_address,omitempty"`
	PaymentMethod   string     `json:"payment_method"`
	PaymentStatus   string     `json:"payment_status"`
	Notes           string     `json:"notes"`
	ShippedAt       *time.Time `json:"shipped_at,omitempty"`
	DeliveredAt     *time.Time `json:"delivered_at,omitempty"`
	CancelledAt     *time.Time `json:"cancelled_at,omitempty"`
	CustomerName    string     `json:"customer_name,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type OrderItem struct {
	ID          string    `json:"id"`
	OrderID     string    `json:"order_id"`
	ProductID   string    `json:"product_id"`
	ProductName string    `json:"product_name"`
	ProductSKU  string    `json:"product_sku"`
	Quantity    int       `json:"quantity"`
	UnitPrice   float64   `json:"unit_price"`
	TotalPrice  float64   `json:"total_price"`
	CreatedAt   time.Time `json:"created_at"`
}
