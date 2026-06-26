package apperrors

import (
	"errors"
	"fmt"
	"net/http"
)

type Code string

const (
	CodeInvalidRequest  Code = "INVALID_REQUEST"
	CodeUnauthorized    Code = "UNAUTHORIZED"
	CodeForbidden       Code = "FORBIDDEN"
	CodeNotFound        Code = "NOT_FOUND"
	CodeConflict        Code = "CONFLICT"
	CodeValidation      Code = "VALIDATION_ERROR"
	CodePaymentFailed   Code = "PAYMENT_FAILED"
	CodeInternal        Code = "INTERNAL_ERROR"
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type AppError struct {
	Code       Code
	Message    string
	Details    []FieldError
	HTTPStatus int
	Err        error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func New(code Code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

func Wrap(code Code, message string, httpStatus int, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Err:        err,
	}
}

func Validation(message string, details ...FieldError) *AppError {
	return &AppError{
		Code:       CodeValidation,
		Message:    message,
		Details:    details,
		HTTPStatus: http.StatusUnprocessableEntity,
	}
}

func IsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

func HTTPStatus(err error) int {
	if appErr, ok := IsAppError(err); ok {
		return appErr.HTTPStatus
	}
	return http.StatusInternalServerError
}
