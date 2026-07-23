package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"backend/dto"
	"backend/models"
	"backend/repositories"
	"backend/utils"
	"backend/validators"
)

var (
	ErrAdminNotFound      = errors.New("admin not found")
	ErrCannotDeleteSelf   = errors.New("cannot delete your own account")
	ErrCannotDeleteSuper  = errors.New("cannot delete another super admin")
	ErrInvalidAdminRole   = errors.New("invalid role: must be admin or super_admin")
	ErrEmailAlreadyExists = errors.New("email already exists")
)

type SuperAdminService interface {
	CreateAdmin(ctx context.Context, req dto.CreateAdminRequest) (models.User, error)
	GetAdminByID(ctx context.Context, id string) (models.User, error)
	ListAdmins(ctx context.Context, query dto.AdminListQuery) (dto.AdminListResponse, error)
	UpdateAdmin(ctx context.Context, id string, req dto.UpdateAdminRequest) (models.User, error)
	UpdateAdminStatus(ctx context.Context, id string, req dto.UpdateAdminStatusRequest) error
	DeleteAdmin(ctx context.Context, id, currentUserId string) error
	GetDashboardStats(ctx context.Context) (dto.DashboardStatsResponse, error)
}

type superAdminService struct {
	superAdminRepo repositories.SuperAdminRepository
	userRepo       repositories.UserRepository
	auditRepo      repositories.AuditRepository
}

func NewSuperAdminService(
	superAdminRepo repositories.SuperAdminRepository,
	userRepo repositories.UserRepository,
	auditRepo repositories.AuditRepository,
) SuperAdminService {
	return &superAdminService{
		superAdminRepo: superAdminRepo,
		userRepo:       userRepo,
		auditRepo:      auditRepo,
	}
}

func (s *superAdminService) CreateAdmin(ctx context.Context, req dto.CreateAdminRequest) (models.User, error) {
	if err := validators.ValidatePassword(req.Password); err != nil {
		return models.User{}, err
	}

	hash, err := utils.GenerateArgon2Hash(req.Password)
	if err != nil {
		return models.User{}, fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := s.superAdminRepo.CreateAdmin(ctx, req.Email, hash, req.FirstName, req.LastName, req.Phone, "admin")
	if err != nil {
		if errors.Is(err, repositories.ErrEmailTaken) {
			return models.User{}, ErrEmailAlreadyExists
		}
		return models.User{}, err
	}

	slog.Info("Admin account created",
		slog.String("admin_id", user.ID),
		slog.String("email", user.Email),
		slog.String("role", user.Role),
	)

	return user, nil
}

func (s *superAdminService) GetAdminByID(ctx context.Context, id string) (models.User, error) {
	user, err := s.superAdminRepo.GetAdminByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrAdminNotFound) {
			return models.User{}, ErrAdminNotFound
		}
		return models.User{}, err
	}
	return user, nil
}

func (s *superAdminService) ListAdmins(ctx context.Context, query dto.AdminListQuery) (dto.AdminListResponse, error) {
	query.SetDefaults()

	offset := (query.Page - 1) * query.Limit
	admins, totalCount, err := s.superAdminRepo.ListAdmins(ctx, query.Search, query.Role, query.Sort, offset, query.Limit)
	if err != nil {
		return dto.AdminListResponse{}, err
	}

	adminResponses := make([]dto.AdminResponse, len(admins))
	for i, admin := range admins {
		adminResponses[i] = dto.AdminResponse{
			ID:        admin.ID,
			Email:     admin.Email,
			FirstName: admin.FirstName,
			LastName:  admin.LastName,
			Role:      admin.Role,
			Phone:     admin.Phone,
			IsActive:  admin.IsActive,
			CreatedAt: admin.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: admin.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	if adminResponses == nil {
		adminResponses = []dto.AdminResponse{}
	}

	return dto.AdminListResponse{
		Admins:     adminResponses,
		TotalCount: totalCount,
		Page:       query.Page,
		Limit:      query.Limit,
		TotalPages: repositories.CalculateTotalPages(totalCount, query.Limit),
	}, nil
}

func (s *superAdminService) UpdateAdmin(ctx context.Context, id string, req dto.UpdateAdminRequest) (models.User, error) {
	user, err := s.superAdminRepo.GetAdminByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrAdminNotFound) {
			return models.User{}, ErrAdminNotFound
		}
		return models.User{}, err
	}

	if user.Role == "super_admin" {
		return models.User{}, errors.New("cannot modify another super admin through this endpoint")
	}

	user, err = s.superAdminRepo.UpdateAdmin(ctx, id, req.FirstName, req.LastName, req.Phone, req.Email)
	if err != nil {
		if errors.Is(err, repositories.ErrEmailTaken) {
			return models.User{}, ErrEmailAlreadyExists
		}
		return models.User{}, err
	}

	slog.Info("Admin updated",
		slog.String("admin_id", user.ID),
		slog.String("email", user.Email),
	)

	return user, nil
}

func (s *superAdminService) UpdateAdminStatus(ctx context.Context, id string, req dto.UpdateAdminStatusRequest) error {
	err := s.superAdminRepo.UpdateAdminStatus(ctx, id, req.IsActive)
	if err != nil {
		if errors.Is(err, repositories.ErrAdminNotFound) {
			return ErrAdminNotFound
		}
		return err
	}

	slog.Info("Admin status updated",
		slog.String("admin_id", id),
		slog.Bool("is_active", req.IsActive),
	)

	return nil
}

func (s *superAdminService) DeleteAdmin(ctx context.Context, id, currentUserId string) error {
	if id == currentUserId {
		return ErrCannotDeleteSelf
	}

	user, err := s.superAdminRepo.GetAdminByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrAdminNotFound) {
			return ErrAdminNotFound
		}
		return err
	}

	if user.Role == "super_admin" {
		return ErrCannotDeleteSuper
	}

	if err := s.superAdminRepo.SoftDeleteAdmin(ctx, id); err != nil {
		return err
	}

	slog.Info("Admin deleted",
		slog.String("admin_id", id),
		slog.String("email", user.Email),
	)

	return nil
}

func (s *superAdminService) GetDashboardStats(ctx context.Context) (dto.DashboardStatsResponse, error) {
	totalUsers, err := s.superAdminRepo.CountAll(ctx)
	if err != nil {
		return dto.DashboardStatsResponse{}, fmt.Errorf("failed to count users: %w", err)
	}

	totalAdmins, err := s.superAdminRepo.CountByRole(ctx, "admin")
	if err != nil {
		return dto.DashboardStatsResponse{}, fmt.Errorf("failed to count admins: %w", err)
	}

	// These queries gracefully return 0 if the table doesn't exist yet
	totalProducts, _ := s.superAdminRepo.CountTableRows(ctx, "products")
	totalCategories, _ := s.superAdminRepo.CountTableRows(ctx, "categories")
	totalOrders, _ := s.superAdminRepo.CountTableRows(ctx, "orders")
	pendingOrders, _ := s.superAdminRepo.CountTableRowsWhere(ctx, "SELECT COUNT(*) FROM orders WHERE status = 'pending'")
	completedOrders, _ := s.superAdminRepo.CountTableRowsWhere(ctx, "SELECT COUNT(*) FROM orders WHERE status = 'completed'")
	totalRevenue, _ := s.superAdminRepo.SumTableColumn(ctx, "SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE status = 'completed'")
	lowStockProducts, _ := s.superAdminRepo.CountTableRowsWhere(ctx, "SELECT COUNT(*) FROM products WHERE stock < 10 AND stock > 0")

	recentUsers, err := s.superAdminRepo.GetRecentUsers(ctx, 5)
	if err != nil {
		slog.Error("Failed to fetch recent users", slog.Any("error", err))
		recentUsers = []models.User{}
	}

	recentAdmins, err := s.superAdminRepo.GetRecentAdmins(ctx, 5)
	if err != nil {
		slog.Error("Failed to fetch recent admins", slog.Any("error", err))
		recentAdmins = []models.User{}
	}

	recentUserResponses := make([]dto.AdminResponse, len(recentUsers))
	for i, u := range recentUsers {
		recentUserResponses[i] = dto.AdminResponse{
			ID:        u.ID,
			Email:     u.Email,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			Role:      u.Role,
			IsActive:  u.IsActive,
			CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: u.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	if recentUserResponses == nil {
		recentUserResponses = []dto.AdminResponse{}
	}

	recentAdminResponses := make([]dto.AdminResponse, len(recentAdmins))
	for i, a := range recentAdmins {
		recentAdminResponses[i] = dto.AdminResponse{
			ID:        a.ID,
			Email:     a.Email,
			FirstName: a.FirstName,
			LastName:  a.LastName,
			Role:      a.Role,
			IsActive:  a.IsActive,
			CreatedAt: a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: a.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	if recentAdminResponses == nil {
		recentAdminResponses = []dto.AdminResponse{}
	}

	return dto.DashboardStatsResponse{
		TotalUsers:       totalUsers,
		TotalAdmins:      totalAdmins,
		TotalProducts:    totalProducts,
		TotalCategories:  totalCategories,
		TotalOrders:      totalOrders,
		PendingOrders:    pendingOrders,
		CompletedOrders:  completedOrders,
		TotalRevenue:     totalRevenue,
		LowStockProducts: lowStockProducts,
		RecentUsers:      recentUserResponses,
		RecentAdmins:     recentAdminResponses,
	}, nil
}
