package service

import (
	"fmt"
	"net/http"
)

// AppError описывает прикладную ошибку сервиса:
// код для клиента, человекочитаемое сообщение, HTTP-статус и вложенная ошибка.
type AppError struct {
	Code    string
	Message string
	Status  int
	Err     error
}

// Error реализует интерфейс error для AppError.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap возвращает вложенную ошибку для поддержки errors.Is/As.
func (e *AppError) Unwrap() error {
	return e.Err
}

// ErrBadRequest конструирует AppError для ошибок валидации или некорректных запросов клиента.
func ErrBadRequest(msg string) *AppError {
	return &AppError{
		Code:    "BAD_REQUEST",
		Message: msg,
		Status:  http.StatusBadRequest,
	}
}

// ErrNotFound конструирует AppError для ситуации, когда ресурс не найден.
func ErrNotFound(msg string) *AppError {
	return &AppError{
		Code:    "NOT_FOUND",
		Message: msg,
		Status:  http.StatusNotFound,
	}
}

// ErrDomain конструирует AppError для доменных конфликтов (например, PR_EXISTS, TEAM_EXISTS).
// Внутри подбирается подходящий HTTP-статус в зависимости от кода.
func ErrDomain(code, msg string) *AppError {
	status := http.StatusConflict
	if code == "TEAM_EXISTS" {
		status = http.StatusBadRequest
	}
	if code == "PR_EXISTS" {
		status = http.StatusConflict
	}
	return &AppError{
		Code:    code,
		Message: msg,
		Status:  status,
	}
}

// IsNotFound помогает определить, соответствует ли ошибка HTTP-статусу 404.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	if app, ok := err.(*AppError); ok {
		return app.Status == http.StatusNotFound
	}
	return false
}
