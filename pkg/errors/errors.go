package errors

import (
	"errors"
	"fmt"
)

type ErrorCode string

const (
	ErrCodeNotFound       ErrorCode = "NOT_FOUND"
	ErrCodeUnauthorized   ErrorCode = "UNAUTHORIZED"
	ErrCodeBadRequest     ErrorCode = "BAD_REQUEST"
	ErrCodeInternal       ErrorCode = "INTERNAL"
	ErrCodeTimeout        ErrorCode = "TIMEOUT"
	ErrCodeRateLimit      ErrorCode = "RATE_LIMIT"
	ErrCodeAnalysis       ErrorCode = "ANALYSIS_ERROR"
	ErrCodeConfiguration  ErrorCode = "CONFIGURATION_ERROR"
)

type AppError struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func New(code ErrorCode, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func Wrap(code ErrorCode, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

func IsNotFound(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == ErrCodeNotFound
	}
	return false
}

func IsUnauthorized(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == ErrCodeUnauthorized
	}
	return false
}

func IsRateLimit(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == ErrCodeRateLimit
	}
	return false
}

func IsConfiguration(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == ErrCodeConfiguration
	}
	return false
}
