package apperror

import (
	"errors"
	"fmt"
)

// AppError は構造化されたアプリケーションエラー。
// cause はサーバーサイドのログ用で、クライアントには返さない。
type AppError struct {
	Code    string
	Message string
	Detail  string
	cause   error
}

func New(code, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func (e *AppError) Error() string {
	s := e.Code + ": " + e.Message
	if e.Detail != "" {
		s += ": " + e.Detail
	}
	if e.cause != nil {
		s += ": " + e.cause.Error()
	}
	return s
}

func (e *AppError) Unwrap() error { return e.cause }

func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

func (e *AppError) Wrap(cause error) *AppError {
	return &AppError{
		Code: e.Code, Message: e.Message,
		Detail: e.Detail, cause: cause,
	}
}

func (e *AppError) WithDetail(detail string) *AppError {
	return &AppError{
		Code: e.Code, Message: e.Message,
		Detail: detail, cause: e.cause,
	}
}

func (e *AppError) Wrapf(cause error, format string, args ...any) *AppError {
	return &AppError{
		Code: e.Code, Message: e.Message,
		Detail: fmt.Sprintf(format, args...), cause: cause,
	}
}

func (e *AppError) Cause() error { return e.cause }

// Extract はエラーチェーンから AppError を取り出す。
// 見つからなければ ErrInternal を返し、生エラーの漏洩を防ぐ。
func Extract(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return ErrInternal
}

// 共通エラー（全サービス共通）
var (
	ErrUnauthorized = New("E0001", "Authentication required.")
	ErrForbidden    = New("E0002", "Permission denied.")
	ErrNotFound     = New("E0003", "Resource not found.")
	ErrConflict     = New("E0004", "Data conflict. Please retry.")
	ErrInvalidInput = New("E0005", "Invalid input.")
	ErrInternal     = New("E0006", "Internal server error.")
)
