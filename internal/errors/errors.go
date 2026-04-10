package errors

import (
	"fmt"
	"net/http"
)

type Code string

const (
	CodeNotFound       Code = "NOT_FOUND"
	CodeInvalidInput   Code = "INVALID_INPUT"
	CodeUnauthorized   Code = "UNAUTHORIZED"
	CodeForbidden      Code = "FORBIDDEN"
	CodeRateLimited    Code = "RATE_LIMITED"
	CodeInternal       Code = "INTERNAL_ERROR"
	CodeServiceUnavail Code = "SERVICE_UNAVAILABLE"
	CodeValidation     Code = "VALIDATION_ERROR"
	CodeConflict       Code = "CONFLICT"
	CodeNotImplemented Code = "NOT_IMPLEMENTED"
)

type AppError struct {
	Code       Code   `json:"code"`
	Message    string `json:"message"`
	Details    string `json:"details,omitempty"`
	Field      string `json:"field,omitempty"`
	HTTPStatus int    `json:"-"`
	Err        error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func (e *AppError) WithField(field string) *AppError {
	e.Field = field
	return e
}

func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

func (e *AppError) WithError(err error) *AppError {
	e.Err = err
	return e
}

type ErrorResponse struct {
	Error ErrorInfo `json:"error"`
}

type ErrorInfo struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
	Details string `json:"details,omitempty"`
}

func NewAppError(code Code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

func NotFound(resource, id string) *AppError {
	return &AppError{
		Code:       CodeNotFound,
		Message:    fmt.Sprintf("%s not found: %s", resource, id),
		HTTPStatus: http.StatusNotFound,
	}
}

func InvalidInput(field, message string) *AppError {
	return &AppError{
		Code:       CodeInvalidInput,
		Message:    message,
		Field:      field,
		HTTPStatus: http.StatusBadRequest,
	}
}

func ValidationError(field, message string) *AppError {
	return &AppError{
		Code:       CodeValidation,
		Message:    message,
		Field:      field,
		HTTPStatus: http.StatusBadRequest,
	}
}

func Unauthorized(message string) *AppError {
	if message == "" {
		message = "Authentication required"
	}
	return &AppError{
		Code:       CodeUnauthorized,
		Message:    message,
		HTTPStatus: http.StatusUnauthorized,
	}
}

func Forbidden(message string) *AppError {
	if message == "" {
		message = "Access denied"
	}
	return &AppError{
		Code:       CodeForbidden,
		Message:    message,
		HTTPStatus: http.StatusForbidden,
	}
}

func RateLimited(retryAfter int) *AppError {
	return &AppError{
		Code:       CodeRateLimited,
		Message:    fmt.Sprintf("Rate limit exceeded. Retry after %d seconds", retryAfter),
		HTTPStatus: http.StatusTooManyRequests,
	}
}

func Internal(err error) *AppError {
	return &AppError{
		Code:       CodeInternal,
		Message:    "An internal error occurred",
		HTTPStatus: http.StatusInternalServerError,
		Err:        err,
	}
}

func ServiceUnavailable(service string) *AppError {
	return &AppError{
		Code:       CodeServiceUnavail,
		Message:    fmt.Sprintf("Service unavailable: %s", service),
		HTTPStatus: http.StatusServiceUnavailable,
	}
}

func Conflict(resource, id string) *AppError {
	return &AppError{
		Code:       CodeConflict,
		Message:    fmt.Sprintf("%s already exists: %s", resource, id),
		HTTPStatus: http.StatusConflict,
	}
}

func NotImplemented(feature string) *AppError {
	return &AppError{
		Code:       CodeNotImplemented,
		Message:    fmt.Sprintf("Feature not implemented: %s", feature),
		HTTPStatus: http.StatusNotImplemented,
	}
}

func (e *AppError) ToResponse() ErrorResponse {
	return ErrorResponse{
		Error: ErrorInfo{
			Code:    e.Code,
			Message: e.Message,
			Field:   e.Field,
			Details: e.Details,
		},
	}
}

func IsNotFound(err error) bool {
	return hasCode(err, CodeNotFound)
}

func IsUnauthorized(err error) bool {
	return hasCode(err, CodeUnauthorized)
}

func IsForbidden(err error) bool {
	return hasCode(err, CodeForbidden)
}

func IsRateLimited(err error) bool {
	return hasCode(err, CodeRateLimited)
}

func hasCode(err error, code Code) bool {
	if err == nil {
		return false
	}
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == code
	}
	return false
}
