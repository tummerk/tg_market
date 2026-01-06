package domain

import (
	"errors"
	"fmt"

	"git.appkode.ru/pub/go/failure"
)

// AppError представляет доменную ошибку приложения.
type AppError struct {
	Code    failure.ErrorCode
	Message string
	cause   error
}

// Error реализует интерфейс error.
func (e *AppError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.cause)
	}
	return e.Message
}

// Unwrap возвращает обёрнутую ошибку для errors.Is/As.
func (e *AppError) Unwrap() error {
	return e.cause
}

// NewError создаёт новую доменную ошибку.
func NewError(code failure.ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// WrapError оборачивает существующую ошибку с доменным контекстом.
func WrapError(err error, code failure.ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		cause:   err,
	}
}

// IsAppError проверяет, является ли ошибка доменной.
func IsAppError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr)
}

// GetCode извлекает код ошибки, если это AppError.
func GetCode(err error) (failure.ErrorCode, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code, true
	}
	return "", false
}
