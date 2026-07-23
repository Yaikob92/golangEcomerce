package dto

// ── Create Admin ──

type CreateAdminRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Phone     string `json:"phone"`
}

// ── Update Admin ──

type UpdateAdminRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
	Email     string `json:"email" binding:"omitempty,email"`
}

// ── Update Admin Status ──

type UpdateAdminStatusRequest struct {
	IsActive bool `json:"is_active"`
}

// ── Admin List Query ──

type AdminListQuery struct {
	Page   int    `form:"page" binding:"omitempty,min=1"`
	Limit  int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Search string `form:"search"`
	Sort   string `form:"sort" binding:"omitempty,oneof=name email created_at -name -email -created_at"`
	Role   string `form:"role" binding:"omitempty,oneof=admin super_admin"`
}

func (q *AdminListQuery) SetDefaults() {
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

// ── Admin Response ──

type AdminResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
	Phone     string `json:"phone"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type AdminListResponse struct {
	Admins      []AdminResponse `json:"admins"`
	TotalCount  int             `json:"total_count"`
	Page        int             `json:"page"`
	Limit       int             `json:"limit"`
	TotalPages  int             `json:"total_pages"`
}

// ── Dashboard Stats ──

type DashboardStatsResponse struct {
	TotalUsers       int              `json:"total_users"`
	TotalAdmins      int              `json:"total_admins"`
	TotalProducts    int              `json:"total_products"`
	TotalCategories  int              `json:"total_categories"`
	TotalOrders      int              `json:"total_orders"`
	PendingOrders    int              `json:"pending_orders"`
	CompletedOrders  int              `json:"completed_orders"`
	TotalRevenue     float64          `json:"total_revenue"`
	LowStockProducts int              `json:"low_stock_products"`
	RecentUsers      []AdminResponse  `json:"recent_users"`
	RecentAdmins     []AdminResponse  `json:"recent_admins"`
}
