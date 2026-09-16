// Package middleware holds the structural request middlewares.
package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"

	"github.com/pixels-two/sow/backend/internal/common"
)

// validate is the shared validator instance. Field names in errors
// come from json tags.
var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return field.Name
		}
		return name
	})
	return v
}

// ValidateRequest binds path, query, header, and body params into a new
// instance of the DTO type. It validates the instance and stores it in
// the echo context under common.EchoContextKeyValidatedDTO. Panics
// unless given a pointer to a struct.
func ValidateRequest(dto any) echo.MiddlewareFunc {
	dtoType := reflect.TypeOf(dto)
	if dtoType == nil || dtoType.Kind() != reflect.Pointer || dtoType.Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("ValidateRequest requires a pointer to struct, got %T", dto))
	}
	elemType := dtoType.Elem()

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			target := reflect.New(elemType).Interface()

			if err := bindAllSources(c, target); err != nil {
				return &common.ValidationError{Details: []common.ValidationDetail{
					{Field: "request", Message: "malformed request payload"},
				}}
			}

			if err := validate.Struct(target); err != nil {
				return toValidationError(err)
			}

			c.Set(common.EchoContextKeyValidatedDTO, target)
			return next(c)
		}
	}
}

// bindAllSources binds every parameter source regardless of HTTP method.
func bindAllSources(c *echo.Context, target any) error {
	if err := echo.BindPathValues(c, target); err != nil {
		return fmt.Errorf("binding path values: %w", err)
	}
	if err := echo.BindQueryParams(c, target); err != nil {
		return fmt.Errorf("binding query params: %w", err)
	}
	if err := echo.BindHeaders(c, target); err != nil {
		return fmt.Errorf("binding headers: %w", err)
	}
	if err := bindBody(c, target); err != nil {
		return fmt.Errorf("binding body: %w", err)
	}
	return nil
}

func bindBody(c *echo.Context, target any) error {
	if _, strict := target.(interface{ StrictJSON() }); !strict {
		return echo.BindBody(c, target)
	}
	mediaType, _, _ := mime.ParseMediaType(c.Request().Header.Get("Content-Type"))
	if mediaType != "application/json" {
		return echo.BindBody(c, target)
	}

	decoder := json.NewDecoder(c.Request().Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decoding json body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("json body contains more than one value")
	}
	return nil
}

func toValidationError(err error) error {
	var fieldErrs validator.ValidationErrors
	if !errors.As(err, &fieldErrs) {
		return &common.ValidationError{Details: []common.ValidationDetail{
			{Field: "request", Message: "invalid request"},
		}}
	}

	details := make([]common.ValidationDetail, 0, len(fieldErrs))
	for _, fe := range fieldErrs {
		details = append(details, common.ValidationDetail{
			Field:   fe.Field(),
			Message: fmt.Sprintf("failed on the %q rule", fe.Tag()),
		})
	}
	return &common.ValidationError{Details: details}
}
