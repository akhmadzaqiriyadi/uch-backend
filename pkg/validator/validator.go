package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func ValidateStruct(s any) []ValidationError {
	var errors []ValidationError
	err := validate.Struct(s)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			field := strings.ToLower(err.Field())
			tag := err.Tag()
			param := err.Param()

			var message string
			switch tag {
			case "required":
				message = fmt.Sprintf("%s is required", field)
			case "email":
				message = fmt.Sprintf("%s must be a valid email address", field)
			case "min":
				message = fmt.Sprintf("%s must be at least %s characters", field, param)
			case "max":
				message = fmt.Sprintf("%s must be at most %s characters", field, param)
			default:
				message = fmt.Sprintf("%s failed on validation rule: %s", field, tag)
			}

			errors = append(errors, ValidationError{
				Field:   field,
				Message: message,
			})
		}
	}
	return errors
}
