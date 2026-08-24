package validator_test

import (
	"testing"

	"gozaq/pkg/validator"
)

type TestUserPayload struct {
	Name     string `validate:"required,min=2"`
	Email    string `validate:"required,email"`
	Password string `validate:"required,min=6"`
}

func TestValidateStruct(t *testing.T) {
	tests := []struct {
		name      string
		payload   TestUserPayload
		wantError bool
	}{
		{
			name: "valid payload",
			payload: TestUserPayload{
				Name:     "Zaq",
				Email:    "zaq@example.com",
				Password: "password123",
			},
			wantError: false,
		},
		{
			name: "missing required fields",
			payload: TestUserPayload{
				Name:     "",
				Email:    "",
				Password: "",
			},
			wantError: true,
		},
		{
			name: "invalid email format",
			payload: TestUserPayload{
				Name:     "Zaq",
				Email:    "invalid-email",
				Password: "password123",
			},
			wantError: true,
		},
		{
			name: "short password",
			payload: TestUserPayload{
				Name:     "Zaq",
				Email:    "zaq@example.com",
				Password: "123",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validator.ValidateStruct(tt.payload)
			if tt.wantError && len(errs) == 0 {
				t.Errorf("expected errors, got none")
			}
			if !tt.wantError && len(errs) > 0 {
				t.Errorf("expected no error, got: %v", errs)
			}
		})
	}
}
