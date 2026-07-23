package utils

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTAccessToken(t *testing.T) {
	secret := "test-secret-key-that-is-long-enough"
	issuer := "test-issuer"
	audience := "test-audience"
	userID := "user-123"
	email := "user@example.com"
	role := "admin"

	token, err := GenerateAccessToken(userID, email, role, secret, issuer, audience)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := ValidateAccessToken(token, secret, issuer, audience)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected UserID %s, got %s", userID, claims.UserID)
	}
	if claims.Email != email {
		t.Errorf("expected Email %s, got %s", email, claims.Email)
	}
	if claims.Role != role {
		t.Errorf("expected Role %s, got %s", role, claims.Role)
	}

	// Test invalid secret
	_, err = ValidateAccessToken(token, "wrong-secret", issuer, audience)
	if err == nil {
		t.Error("expected validation to fail with wrong secret")
	}

	// Test invalid issuer
	_, err = ValidateAccessToken(token, secret, "wrong-issuer", audience)
	if err == nil {
		t.Error("expected validation to fail with wrong issuer")
	}

	// Test invalid audience
	_, err = ValidateAccessToken(token, secret, issuer, "wrong-audience")
	if err == nil {
		t.Error("expected validation to fail with wrong audience")
	}
}

func TestExpiredToken(t *testing.T) {
	secret := "test-secret"
	claims := AccessTokenClaims{
		UserID: "123",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)), // Expired 1 min ago
			Issuer:    "issuer",
			Audience:  jwt.ClaimStrings{"aud"},
		},
	}
	tokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := tokenObj.SignedString([]byte(secret))

	_, err := ValidateAccessToken(tokenStr, secret, "issuer", "aud")
	if err != ErrExpiredToken {
		t.Errorf("expected ErrExpiredToken, got %v", err)
	}
}
