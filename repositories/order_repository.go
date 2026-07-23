package repositories

import (
	"context"
	"fmt"
	"strings"

	"backend/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepository interface {
	List(ctx context.Context, search, status, sort string, offset, limit int) ([]models.Order, int, error)
	GetByID(ctx context.Context, id string) (models.Order, error)
	GetItemsByOrderID(ctx context.Context, orderID string) ([]models.OrderItem, error)
}

type postgresOrderRepository struct {
	pool *pgxpool.Pool
}

func NewOrderRepository(pool *pgxpool.Pool) OrderRepository {
	return &postgresOrderRepository{pool: pool}
}

func (r *postgresOrderRepository) List(ctx context.Context, search, status, sort string, offset, limit int) ([]models.Order, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "o.deleted_at IS NULL")

	if status != "" {
		conditions = append(conditions, fmt.Sprintf("o.status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}

	if search != "" {
		conditions = append(conditions, fmt.Sprintf("(LOWER(o.order_number) LIKE $%d OR LOWER(u.first_name || ' ' || u.last_name) LIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+strings.ToLower(search)+"%")
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM orders o LEFT JOIN users u ON u.id = o.user_id %s", where)
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count orders: %w", err)
	}

	orderBy := mapOrderSortToSQL(sort)

	dataQuery := fmt.Sprintf(`
		SELECT o.id, o.user_id, o.order_number, o.status, o.subtotal, o.shipping_cost,
			o.tax, o.discount, o.total_amount, o.currency, o.payment_method, o.payment_status,
			COALESCE(u.first_name || ' ' || u.last_name, 'Unknown'),
			o.created_at, o.updated_at
		FROM orders o
		LEFT JOIN users u ON u.id = o.user_id
		%s ORDER BY %s LIMIT $%d OFFSET $%d`,
		where, orderBy, argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list orders: %w", err)
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		err := rows.Scan(
			&o.ID, &o.UserID, &o.OrderNumber, &o.Status, &o.Subtotal, &o.ShippingCost,
			&o.Tax, &o.Discount, &o.TotalAmount, &o.Currency, &o.PaymentMethod, &o.PaymentStatus,
			&o.CustomerName, &o.CreatedAt, &o.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, o)
	}

	return orders, total, nil
}

func (r *postgresOrderRepository) GetByID(ctx context.Context, id string) (models.Order, error) {
	var o models.Order
	err := r.pool.QueryRow(ctx,
		`SELECT o.id, o.user_id, o.order_number, o.status, o.subtotal, o.shipping_cost,
			o.tax, o.discount, o.total_amount, o.currency, o.payment_method, o.payment_status,
			COALESCE(u.first_name || ' ' || u.last_name, 'Unknown'),
			o.created_at, o.updated_at
		FROM orders o
		LEFT JOIN users u ON u.id = o.user_id
		WHERE o.id = $1 AND o.deleted_at IS NULL`, id,
	).Scan(
		&o.ID, &o.UserID, &o.OrderNumber, &o.Status, &o.Subtotal, &o.ShippingCost,
		&o.Tax, &o.Discount, &o.TotalAmount, &o.Currency, &o.PaymentMethod, &o.PaymentStatus,
		&o.CustomerName, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return models.Order{}, fmt.Errorf("order not found")
		}
		return models.Order{}, err
	}
	return o, nil
}

func (r *postgresOrderRepository) GetItemsByOrderID(ctx context.Context, orderID string) ([]models.OrderItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, order_id, product_id, product_name, COALESCE(product_sku, ''),
			quantity, unit_price, total_price, created_at
		FROM order_items WHERE order_id = $1 ORDER BY created_at ASC`, orderID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list order items: %w", err)
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		err := rows.Scan(
			&item.ID, &item.OrderID, &item.ProductID, &item.ProductName, &item.ProductSKU,
			&item.Quantity, &item.UnitPrice, &item.TotalPrice, &item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order item: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func mapOrderSortToSQL(sort string) string {
	switch sort {
	case "total_amount":
		return "o.total_amount ASC"
	case "-total_amount":
		return "o.total_amount DESC"
	case "created_at":
		return "o.created_at ASC"
	case "-created_at", "":
		return "o.created_at DESC"
	case "status":
		return "o.status ASC"
	case "-status":
		return "o.status DESC"
	default:
		return "o.created_at DESC"
	}
}
