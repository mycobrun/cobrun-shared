// Package validation provides input validation utilities.
package validation

import (
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	validate *validator.Validate
	once     sync.Once
)

// GetValidator returns the singleton validator instance.
func GetValidator() *validator.Validate {
	once.Do(func() {
		validate = validator.New()

		// Use JSON tag names for error messages
		validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})

		// Register custom validations
		registerCustomValidations(validate)
	})

	return validate
}

func registerCustomValidations(v *validator.Validate) {
	// Phone number validation (E.164 format)
	v.RegisterValidation("phone", validatePhone)

	// Latitude validation
	v.RegisterValidation("latitude", validateLatitude)

	// Longitude validation
	v.RegisterValidation("longitude", validateLongitude)

	// User type validation
	v.RegisterValidation("user_type", validateUserType)

	// Trip status validation
	v.RegisterValidation("trip_status", validateTripStatus)

	// Currency code validation (ISO 4217)
	v.RegisterValidation("currency", validateCurrency)

	// UUID validation
	v.RegisterValidation("uuid4", validateUUID4)
}

// Phone validates E.164 phone numbers.
var phoneRegex = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

func validatePhone(fl validator.FieldLevel) bool {
	return phoneRegex.MatchString(fl.Field().String())
}

// Latitude validates latitude values (-90 to 90).
func validateLatitude(fl validator.FieldLevel) bool {
	lat := fl.Field().Float()
	return lat >= -90 && lat <= 90
}

// Longitude validates longitude values (-180 to 180).
func validateLongitude(fl validator.FieldLevel) bool {
	lng := fl.Field().Float()
	return lng >= -180 && lng <= 180
}

// User types.
var validUserTypes = map[string]bool{
	"rider":  true,
	"driver": true,
	"admin":  true,
}

func validateUserType(fl validator.FieldLevel) bool {
	return validUserTypes[fl.Field().String()]
}

// Trip statuses.
var validTripStatuses = map[string]bool{
	"requested":   true,
	"accepted":    true,
	"arriving":    true,
	"in_progress": true,
	"completed":   true,
	"cancelled":   true,
}

func validateTripStatus(fl validator.FieldLevel) bool {
	return validTripStatuses[fl.Field().String()]
}

// Currency codes (common ones).
var validCurrencies = map[string]bool{
	"USD": true,
	"EUR": true,
	"GBP": true,
	"CAD": true,
	"AUD": true,
	"MXN": true,
}

func validateCurrency(fl validator.FieldLevel) bool {
	return validCurrencies[fl.Field().String()]
}

// UUID4 validation.
var uuid4Regex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

func validateUUID4(fl validator.FieldLevel) bool {
	return uuid4Regex.MatchString(fl.Field().String())
}

// Validate validates a struct and returns validation errors.
func Validate(s interface{}) error {
	return GetValidator().Struct(s)
}

// ValidationError represents a single validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors []ValidationError

// Error implements the error interface.
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, e := range ve {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(e.Field)
		sb.WriteString(": ")
		sb.WriteString(e.Message)
	}
	return sb.String()
}

// ParseValidationErrors converts validator.ValidationErrors to our format.
func ParseValidationErrors(err error) ValidationErrors {
	if err == nil {
		return nil
	}

	var validationErrors ValidationErrors

	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, e := range ve {
			validationErrors = append(validationErrors, ValidationError{
				Field:   e.Field(),
				Message: getErrorMessage(e),
			})
		}
	}

	return validationErrors
}

func getErrorMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "phone":
		return "must be a valid phone number in E.164 format"
	case "latitude":
		return "must be a valid latitude (-90 to 90)"
	case "longitude":
		return "must be a valid longitude (-180 to 180)"
	case "min":
		return "must be at least " + e.Param()
	case "max":
		return "must be at most " + e.Param()
	case "len":
		return "must be exactly " + e.Param() + " characters"
	case "uuid4":
		return "must be a valid UUID v4"
	case "user_type":
		return "must be one of: rider, driver, admin"
	case "trip_status":
		return "must be a valid trip status"
	case "currency":
		return "must be a valid currency code"
	case "oneof":
		return "must be one of: " + e.Param()
	case "gt":
		return "must be greater than " + e.Param()
	case "gte":
		return "must be greater than or equal to " + e.Param()
	case "lt":
		return "must be less than " + e.Param()
	case "lte":
		return "must be less than or equal to " + e.Param()
	default:
		return "is invalid"
	}
}

// ValidateVar validates a single variable.
func ValidateVar(field interface{}, tag string) error {
	return GetValidator().Var(field, tag)
}

// Validator wraps the go-playground validator for easier use.
type Validator struct {
	v *validator.Validate
}

// New creates a new Validator instance.
func New() *Validator {
	return &Validator{v: GetValidator()}
}

// Struct validates a struct.
func (v *Validator) Struct(s interface{}) error {
	return v.v.Struct(s)
}

// Var validates a single variable.
func (v *Validator) Var(field interface{}, tag string) error {
	return v.v.Var(field, tag)
}
