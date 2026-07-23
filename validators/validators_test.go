package validators

import (
	"testing"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{
			name:     "valid password",
			password: "SecurePass123!",
			wantErr:  nil,
		},
		{
			name:     "too short",
			password: "Sh123!",
			wantErr:  ErrPasswordTooShort,
		},
		{
			name:     "no uppercase",
			password: "securepass123!",
			wantErr:  ErrPasswordNoUpper,
		},
		{
			name:     "no lowercase",
			password: "SECUREPASS123!",
			wantErr:  ErrPasswordNoLower,
		},
		{
			name:     "no digit",
			password: "SecurePass!",
			wantErr:  ErrPasswordNoDigit,
		},
		{
			name:     "no special",
			password: "SecurePass123",
			wantErr:  ErrPasswordNoSpecial,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if err != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
