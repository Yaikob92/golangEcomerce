package services

import (
	"context"
	"fmt"
	"strings"

	"backend/dto"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CustomerService interface {
	List(ctx context.Context, search string, page, limit int) (dto.CustomerListResponse, error)
}

type customerService struct {
	pool *pgxpool.Pool
}

func NewCustomerService(pool *pgxpool.Pool) CustomerService {
	return &customerService{pool: pool}
}

func (s *customerService) List(ctx context.Context, search string, page, limit int) (dto.CustomerListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "u.deleted_at IS NULL")
	conditions = append(conditions, "u.role = 'customer'")

	if search != "" {
		conditions = append(conditions, fmt.Sprintf("(LOWER(u.first_name) LIKE $%d OR LOWER(u.last_name) LIKE $%d OR LOWER(u.email) LIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+strings.ToLower(search)+"%")
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users u %s", where)
	err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return dto.CustomerListResponse{}, fmt.Errorf("failed to count customers: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT u.id, COALESCE(u.first_name, ''), COALESCE(u.last_name, ''), u.email,
			COALESCE(u.phone, ''), u.created_at,
			COALESCE(order_stats.order_count, 0),
			COALESCE(order_stats.total_spent, 0)
		FROM users u
		LEFT JOIN (
			SELECT user_id, COUNT(*) AS order_count, COALESCE(SUM(total_amount), 0) AS total_spent
			FROM orders WHERE deleted_at IS NULL GROUP BY user_id
		) order_stats ON order_stats.user_id = u.id
		%s ORDER BY u.created_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := s.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return dto.CustomerListResponse{}, fmt.Errorf("failed to list customers: %w", err)
	}
	defer rows.Close()

	var customers []dto.CustomerResponse
	for rows.Next() {
		var c dto.CustomerResponse
		err := rows.Scan(
			&c.ID, &c.FirstName, &c.LastName, &c.Email,
			&c.Phone, &c.CreatedAt, &c.OrdersCount, &c.TotalSpent,
		)
		if err != nil {
			return dto.CustomerListResponse{}, fmt.Errorf("failed to scan customer: %w", err)
		}
		customers = append(customers, c)
	}
	if customers == nil {
		customers = []dto.CustomerResponse{}
	}

	totalPages := 1
	if limit > 0 {
		totalPages = total / limit
		if total%limit != 0 {
			totalPages++
		}
	}

	return dto.CustomerListResponse{
		Customers:  customers,
		TotalCount: total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}
