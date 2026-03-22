package auth

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	CreateUser(ctx context.Context, name, email, passwordHash, role string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	StoreRefreshToken(ctx context.Context, userID string, token string, expiresAt time.Time) error
	GetRefreshTokenUserID(ctx context.Context, token string) (string, error)
	DeleteRefreshToken(ctx context.Context, token string) error
}

type postgresRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) CreateUser(ctx context.Context, name, email, passwordHash, role string) (*User, error) {
	if role == "" {
		role = "user"
	}

	user := User{
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
	}

	err := r.db.WithContext(ctx).Create(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *postgresRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

func (r *postgresRepository) GetUserByID(ctx context.Context, id string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

func (r *postgresRepository) StoreRefreshToken(ctx context.Context, userID string, token string, expiresAt time.Time) error {
	refreshToken := RefreshToken{
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
	}
	return r.db.WithContext(ctx).Create(&refreshToken).Error
}

func (r *postgresRepository) GetRefreshTokenUserID(ctx context.Context, token string) (string, error) {
	var refreshToken RefreshToken
	err := r.db.WithContext(ctx).Where("token = ?", token).First(&refreshToken).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("token not found")
		}
		return "", err
	}

	if time.Now().After(refreshToken.ExpiresAt) {
		// Clean up expired token
		_ = r.DeleteRefreshToken(ctx, token)
		return "", errors.New("token expired")
	}

	return refreshToken.UserID, nil
}

func (r *postgresRepository) DeleteRefreshToken(ctx context.Context, token string) error {
	return r.db.WithContext(ctx).Where("token = ?", token).Delete(&RefreshToken{}).Error
}
