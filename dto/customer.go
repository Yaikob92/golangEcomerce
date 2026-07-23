package dto

// ── Customer ──

type CustomerResponse struct {
	ID          string  `json:"id"`
	FirstName   string  `json:"first_name"`
	LastName    string  `json:"last_name"`
	Email       string  `json:"email"`
	Phone       string  `json:"phone"`
	OrdersCount int     `json:"orders_count"`
	TotalSpent  float64 `json:"total_spent"`
	CreatedAt   string  `json:"created_at"`
}

type CustomerListResponse struct {
	Customers  []CustomerResponse `json:"customers"`
	TotalCount int                `json:"total_count"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	TotalPages int                `json:"total_pages"`
}
