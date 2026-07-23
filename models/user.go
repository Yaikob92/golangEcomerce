package models

import "time"

type User struct {
	ID                  string     `json:"id"`
	Email               string     `json:"email"`
	PasswordHash        string     `json:"-"`
	FirstName           string     `json:"first_name"`
	LastName            string     `json:"last_name"`
	Role                string     `json:"role"`
	IsVerified          bool       `json:"is_verified"`
	FailedLoginAttempts int        `json:"failed_login_attempts"`
	LockoutUntil        *time.Time `json:"lockout_until"`
	Phone               string     `json:"phone"`
	CompanyName         string     `json:"company_name"`
	Address             string     `json:"address"`
	ProfilePictureURL   string     `json:"profile_picture_url"`
	PreferredLanguage   string     `json:"preferred_language"`
	Timezone            string     `json:"timezone"`
	EmailNotifications  bool       `json:"email_notifications"`
	SMSNotifications    bool       `json:"sms_notifications"`
	MarketingEmails     bool       `json:"marketing_emails"`
	IsActive            bool       `json:"is_active"`
	DeletedAt           *time.Time `json:"deleted_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}
