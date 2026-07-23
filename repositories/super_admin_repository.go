package repositories

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"backend/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAdminNotFound       = errors.New("admin not found")
	ErrCannotDeleteSelf    = errors.New("cannot delete your own account")
	ErrCannotDeleteSuper   = errors.New("cannot delete another super admin")
	ErrSuperAdminRequired  = errors.New("super admin role required")
)

type SuperAdminRepository interface {
	CreateAdmin(ctx context.Context, email, passwordHash, firstName, lastName, phone, role string) (models.User, error)
	GetAdminByID(ctx context.Context, id string) (models.User, error)
	ListAdmins(ctx context.Context, search, role, sort string, offset, limit int) ([]models.User, int, error)
	UpdateAdmin(ctx context.Context, id, firstName, lastName, phone, email string) (models.User, error)
	UpdateAdminStatus(ctx context.Context, id string, isActive bool) error
	SoftDeleteAdmin(ctx context.Context, id string) error
	CountByRole(ctx context.Context, role string) (int, error)
	CountAll(ctx context.Context) (int, error)
	CountActive(ctx context.Context) (int, error)
	GetRecentUsers(ctx context.Context, limit int) ([]models.User, error)
	GetRecentAdmins(ctx context.Context, limit int) ([]models.User, error)
	CountTableRows(ctx context.Context, table string) (int, error)
	CountTableRowsWhere(ctx context.Context, query string, args ...interface{}) (int, error)
	SumTableColumn(ctx context.Context, query string, args ...interface{}) (float64, error)
}

type postgresSuperAdminRepository struct {
	pool *pgxpool.Pool
}

func NewSuperAdminRepository(pool *pgxpool.Pool) SuperAdminRepository {
	return &postgresSuperAdminRepository{pool: pool}
}

const userSelectCols = `
	id, email, password_hash, first_name, last_name, role, is_verified,
	failed_login_attempts, lockout_until,
	COALESCE(phone, ''), COALESCE(company_name, ''), COALESCE(address, ''),
	COALESCE(profile_picture_url, ''), COALESCE(preferred_language, 'en'),
	COALESCE(timezone, 'UTC'),
	COALESCE(email_notifications, true), COALESCE(sms_notifications, false),
	COALESCE(marketing_emails, false),
	COALESCE(is_active, true), deleted_at,
	created_at, updated_at
`

func scanUser(row pgx.Row) (models.User, error) {
	var u models.User
	err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName,
		&u.Role, &u.IsVerified, &u.FailedLoginAttempts, &u.LockoutUntil,
		&u.Phone, &u.CompanyName, &u.Address, &u.ProfilePictureURL,
		&u.PreferredLanguage, &u.Timezone,
		&u.EmailNotifications, &u.SMSNotifications, &u.MarketingEmails,
		&u.IsActive, &u.DeletedAt,
		&u.CreatedAt, &u.UpdatedAt,
	)
	return u, err
}

func (r *postgresSuperAdminRepository) CreateAdmin(ctx context.Context, email, passwordHash, firstName, lastName, phone, role string) (models.User, error) {
	var u models.User
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, first_name, last_name, phone, role, is_verified, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6, TRUE, TRUE)
		 RETURNING `+userSelectCols,
		email, passwordHash, firstName, lastName, phone, role,
	).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName,
		&u.Role, &u.IsVerified, &u.FailedLoginAttempts, &u.LockoutUntil,
		&u.Phone, &u.CompanyName, &u.Address, &u.ProfilePictureURL,
		&u.PreferredLanguage, &u.Timezone,
		&u.EmailNotifications, &u.SMSNotifications, &u.MarketingEmails,
		&u.IsActive, &u.DeletedAt,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if isDuplicateKeyError(err) {
			return models.User{}, ErrEmailTaken
		}
		return models.User{}, fmt.Errorf("failed to create admin: %w", err)
	}
	return u, nil
}

func (r *postgresSuperAdminRepository) GetAdminByID(ctx context.Context, id string) (models.User, error) {
	var u models.User
	err := r.pool.QueryRow(ctx,
		`SELECT `+userSelectCols+`
		 FROM users WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName,
		&u.Role, &u.IsVerified, &u.FailedLoginAttempts, &u.LockoutUntil,
		&u.Phone, &u.CompanyName, &u.Address, &u.ProfilePictureURL,
		&u.PreferredLanguage, &u.Timezone,
		&u.EmailNotifications, &u.SMSNotifications, &u.MarketingEmails,
		&u.IsActive, &u.DeletedAt,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, ErrAdminNotFound
		}
		return models.User{}, err
	}
	return u, nil
}

func (r *postgresSuperAdminRepository) ListAdmins(ctx context.Context, search, role, sort string, offset, limit int) ([]models.User, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	// Exclude soft-deleted
	conditions = append(conditions, fmt.Sprintf("deleted_at IS NULL"))

	// Role filter
	if role != "" {
		conditions = append(conditions, fmt.Sprintf("role = $%d", argIdx))
		args = append(args, role)
		argIdx++
	}

	// Search filter
	if search != "" {
		conditions = append(conditions, fmt.Sprintf("(LOWER(email) LIKE $%d OR LOWER(first_name) LIKE $%d OR LOWER(last_name) LIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+strings.ToLower(search)+"%")
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users %s", where)
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count admins: %w", err)
	}

	// Sort mapping
	orderBy := mapSortToSQL(sort)

	// Data query
	dataQuery := fmt.Sprintf(
		`SELECT %s FROM users %s ORDER BY %s LIMIT $%d OFFSET $%d`,
		userSelectCols, where, orderBy, argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list admins: %w", err)
	}
	defer rows.Close()

	var admins []models.User
	for rows.Next() {
		var u models.User
		err := rows.Scan(
			&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName,
			&u.Role, &u.IsVerified, &u.FailedLoginAttempts, &u.LockoutUntil,
			&u.Phone, &u.CompanyName, &u.Address, &u.ProfilePictureURL,
			&u.PreferredLanguage, &u.Timezone,
			&u.EmailNotifications, &u.SMSNotifications, &u.MarketingEmails,
			&u.IsActive, &u.DeletedAt,
			&u.CreatedAt, &u.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan admin: %w", err)
		}
		admins = append(admins, u)
	}

	return admins, total, nil
}

func (r *postgresSuperAdminRepository) UpdateAdmin(ctx context.Context, id, firstName, lastName, phone, email string) (models.User, error) {
	var u models.User
	err := r.pool.QueryRow(ctx,
		`UPDATE users SET
			first_name = COALESCE(NULLIF($1, ''), first_name),
			last_name = COALESCE(NULLIF($2, ''), last_name),
			phone = $3,
			email = COALESCE(NULLIF($4, ''), email),
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = $5 AND deleted_at IS NULL
		 RETURNING `+userSelectCols,
		firstName, lastName, phone, email, id,
	).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName,
		&u.Role, &u.IsVerified, &u.FailedLoginAttempts, &u.LockoutUntil,
		&u.Phone, &u.CompanyName, &u.Address, &u.ProfilePictureURL,
		&u.PreferredLanguage, &u.Timezone,
		&u.EmailNotifications, &u.SMSNotifications, &u.MarketingEmails,
		&u.IsActive, &u.DeletedAt,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, ErrAdminNotFound
		}
		if isDuplicateKeyError(err) {
			return models.User{}, ErrEmailTaken
		}
		return models.User{}, fmt.Errorf("failed to update admin: %w", err)
	}
	return u, nil
}

func (r *postgresSuperAdminRepository) UpdateAdminStatus(ctx context.Context, id string, isActive bool) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE users SET is_active = $1, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $2 AND deleted_at IS NULL AND role != 'super_admin'`,
		isActive, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update admin status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrAdminNotFound
	}
	return nil
}

func (r *postgresSuperAdminRepository) SoftDeleteAdmin(ctx context.Context, id string) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE users SET deleted_at = CURRENT_TIMESTAMP, is_active = FALSE, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $1 AND deleted_at IS NULL AND role != 'super_admin'`,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to delete admin: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrAdminNotFound
	}
	return nil
}

func (r *postgresSuperAdminRepository) CountByRole(ctx context.Context, role string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM users WHERE role = $1 AND deleted_at IS NULL",
		role,
	).Scan(&count)
	return count, err
}

func (r *postgresSuperAdminRepository) CountAll(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM users WHERE deleted_at IS NULL",
	).Scan(&count)
	return count, err
}

func (r *postgresSuperAdminRepository) CountActive(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND is_active = TRUE",
	).Scan(&count)
	return count, err
}

func (r *postgresSuperAdminRepository) GetRecentUsers(ctx context.Context, limit int) ([]models.User, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+userSelectCols+`
		 FROM users WHERE deleted_at IS NULL AND role = 'customer'
		 ORDER BY created_at DESC LIMIT $1`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		u, err := scanUserRow(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *postgresSuperAdminRepository) GetRecentAdmins(ctx context.Context, limit int) ([]models.User, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+userSelectCols+`
		 FROM users WHERE deleted_at IS NULL AND role IN ('admin', 'super_admin')
		 ORDER BY created_at DESC LIMIT $1`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var admins []models.User
	for rows.Next() {
		u, err := scanUserRow(rows)
		if err != nil {
			return nil, err
		}
		admins = append(admins, u)
	}
	return admins, nil
}

func scanUserRow(rows pgx.Rows) (models.User, error) {
	var u models.User
	err := rows.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName,
		&u.Role, &u.IsVerified, &u.FailedLoginAttempts, &u.LockoutUntil,
		&u.Phone, &u.CompanyName, &u.Address, &u.ProfilePictureURL,
		&u.PreferredLanguage, &u.Timezone,
		&u.EmailNotifications, &u.SMSNotifications, &u.MarketingEmails,
		&u.IsActive, &u.DeletedAt,
		&u.CreatedAt, &u.UpdatedAt,
	)
	return u, err
}

func mapSortToSQL(sort string) string {
	switch sort {
	case "name":
		return "first_name ASC, last_name ASC"
	case "-name":
		return "first_name DESC, last_name DESC"
	case "email":
		return "email ASC"
	case "-email":
		return "email DESC"
	case "created_at":
		return "created_at ASC"
	case "-created_at", "":
		return "created_at DESC"
	default:
		return "created_at DESC"
	}
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "23505") || strings.Contains(errStr, "unique constraint")
}

// CountTableRows counts rows in a table, returns 0 if table doesn't exist.
func (r *postgresSuperAdminRepository) CountTableRows(ctx context.Context, table string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
	if err != nil {
		// If the table doesn't exist, return 0
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "relation") {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

// CountTableRowsWhere counts rows with a WHERE clause, returns 0 if table doesn't exist.
func (r *postgresSuperAdminRepository) CountTableRowsWhere(ctx context.Context, query string, args ...interface{}) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "relation") {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

// SumTableColumn sums a column, returns 0 if table doesn't exist.
func (r *postgresSuperAdminRepository) SumTableColumn(ctx context.Context, query string, args ...interface{}) (float64, error) {
	var sum float64
	err := r.pool.QueryRow(ctx, query, args...).Scan(&sum)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "relation") {
			return 0, nil
		}
		return 0, err
	}
	return sum, nil
}

// Helper to calculate total pages
func CalculateTotalPages(totalCount, limit int) int {
	return int(math.Ceil(float64(totalCount) / float64(limit)))
}
