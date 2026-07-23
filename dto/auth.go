package dto

// ErrorDetail represents an error code detail.
type ErrorDetail struct {
	Code string `json:"code" example:"VALIDATION_ERROR"`
}

// ErrorResponse is the standardized error structure.
type ErrorResponse struct {
	Success bool        `json:"success" example:"false"`
	Message string      `json:"message" example:"An error occurred."`
	Error   ErrorDetail `json:"error"`
}

// SuccessResponse is the standardized success structure.
type SuccessResponse struct {
	Success bool        `json:"success" example:"true"`
	Message string      `json:"message,omitempty" example:"Operation completed successfully."`
	Data    interface{} `json:"data,omitempty"`
}

// RegisterRequest represents registration data.
type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email" example:"user@example.com"`
	Password  string `json:"password" binding:"required" example:"SecurePass1!"`
	FirstName string `json:"first_name" binding:"required" example:"John"`
	LastName  string `json:"last_name" binding:"required" example:"Doe"`
}

// LoginRequest represents login data.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required" example:"SecurePass1!"`
}

// VerifyEmailRequest represents verification data.
type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required" example:"abc123def456"`
}

// ResendVerificationRequest represents resending verification data.
type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required,email" example:"user@example.com"`
}

// ForgotPasswordRequest represents forgot password data.
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email" example:"user@example.com"`
}

// ResetPasswordRequest represents reset password data.
type ResetPasswordRequest struct {
	Token    string `json:"token" binding:"required" example:"reset-token-abc123"`
	Password string `json:"password" binding:"required" example:"NewSecurePass1!"`
}

// SessionResponse represents active device sessions.
type SessionResponse struct {
	ID              string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	DeviceName      string `json:"device_name" example:"Chrome on Linux"`
	Browser         string `json:"browser" example:"Chrome 120"`
	OperatingSystem string `json:"operating_system" example:"Linux"`
	IPAddress       string `json:"ip_address" example:"192.168.1.1"`
	LastActive      string `json:"last_active" example:"2025-01-15T10:30:00Z"`
	IsCurrent       bool   `json:"is_current" example:"true"`
}

// UpdateProfileRequest represents profile update data.
type UpdateProfileRequest struct {
	FirstName         string `json:"first_name" binding:"required" example:"John"`
	LastName          string `json:"last_name" binding:"required" example:"Doe"`
	Phone             string `json:"phone" example:"+1234567890"`
	CompanyName       string `json:"company_name" example:"Acme Corp"`
	Address           string `json:"address" example:"123 Main St, New York, NY 10001"`
	PreferredLanguage string `json:"preferred_language" example:"en"`
	Timezone          string `json:"timezone" example:"America/New_York"`
}

// ChangePasswordRequest represents password change data.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required" example:"CurrentPass1!"`
	NewPassword     string `json:"new_password" binding:"required" example:"NewSecurePass1!"`
}

// UpdateNotificationPreferencesRequest represents notification settings.
type UpdateNotificationPreferencesRequest struct {
	EmailNotifications bool `json:"email_notifications"`
	SMSNotifications   bool `json:"sms_notifications"`
	MarketingEmails    bool `json:"marketing_emails"`
}
