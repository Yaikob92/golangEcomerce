package models

import "time"

type AuditLog struct {
	ID              string    `json:"id"`
	UserID          *string   `json:"user_id"`
	Event           string    `json:"event"`
	IPAddress       string    `json:"ip_address"`
	DeviceName      string    `json:"device_name"`
	Browser         string    `json:"browser"`
	OperatingSystem string    `json:"operating_system"`
	Timestamp       time.Time `json:"timestamp"`
}
