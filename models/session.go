package models

import "time"

type Session struct {
	ID               string     `json:"id"`
	UserID           string     `json:"user_id"`
	RefreshTokenHash string     `json:"-"`
	DeviceName       string     `json:"device_name"`
	Browser          string     `json:"browser"`
	OperatingSystem  string     `json:"operating_system"`
	IPAddress        string     `json:"ip_address"`
	UserAgent        string     `json:"user_agent"`
	CreatedAt        time.Time  `json:"created_at"`
	LastUsedAt       time.Time  `json:"last_used_at"`
	ExpiresAt        time.Time  `json:"expires_at"`
	RevokedAt        *time.Time `json:"revoked_at"`
}
