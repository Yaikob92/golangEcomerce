package validators

import (
	"errors"
	"regexp"
)

var (
	ErrPasswordTooShort = errors.New("password must be at least 8 characters long")
	ErrPasswordNoUpper  = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNoLower  = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNoDigit  = errors.New("password must contain at least one number")
	ErrPasswordNoSpecial = errors.New("password must contain at least one special character")
)

// ValidatePassword checks password strength rules.
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}

	var (
		hasUpper   = regexp.MustCompile(`[A-Z]`).MatchString(password)
		hasLower   = regexp.MustCompile(`[a-z]`).MatchString(password)
		hasDigit   = regexp.MustCompile(`[0-9]`).MatchString(password)
		hasSpecial = regexp.MustCompile(`[!@#\$%\^&\*\(\)_\+\-=\[\]\{\};':",\./<>\?~\\|]`).MatchString(password)
	)

	if !hasUpper {
		return ErrPasswordNoUpper
	}
	if !hasLower {
		return ErrPasswordNoLower
	}
	if !hasDigit {
		return ErrPasswordNoDigit
	}
	if !hasSpecial {
		return ErrPasswordNoSpecial
	}

	return nil
}
