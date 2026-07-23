package dto

// ── Order ──

type OrderResponse struct {
	ID            string              `json:"id"`
	OrderNumber   string              `json:"order_number"`
	CustomerName  string              `json:"customer_name,omitempty"`
	Status        string              `json:"status"`
	Subtotal      float64             `json:"subtotal,omitempty"`
	ShippingCost  float64             `json:"shipping_cost,omitempty"`
	Tax           float64             `json:"tax,omitempty"`
	Discount      float64             `json:"discount,omitempty"`
	TotalAmount   float64             `json:"total_amount"`
	PaymentMethod string              `json:"payment_method,omitempty"`
	PaymentStatus string              `json:"payment_status,omitempty"`
	CreatedAt     string              `json:"created_at"`
	UpdatedAt     string              `json:"updated_at,omitempty"`
	Items         []OrderItemResponse `json:"items,omitempty"`
}

type OrderItemResponse struct {
	ID          string  `json:"id"`
	ProductName string  `json:"product_name"`
	ProductSKU  string  `json:"product_sku"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	TotalPrice  float64 `json:"total_price"`
}

type OrderListResponse struct {
	Orders     []OrderResponse `json:"orders"`
	TotalCount int             `json:"total_count"`
	Page       int             `json:"page"`
	Limit      int             `json:"limit"`
	TotalPages int             `json:"total_pages"`
}
