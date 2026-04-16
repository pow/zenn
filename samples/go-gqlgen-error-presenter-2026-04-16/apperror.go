package errorpresenter

import "fmt"

// Code はアプリケーションエラーの種別を表す。
type Code string

const (
	CodeNotFound   Code = "NOT_FOUND"
	CodeForbidden  Code = "FORBIDDEN"
	CodeBadRequest Code = "BAD_REQUEST"
	CodeInternal   Code = "INTERNAL"
)

// AppError はアプリケーション層のエラーを表す構造体。
type AppError struct {
	Code    Code
	Message string
	Err     error // ラップ元のエラー（ログ用）
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Err }

func NewNotFound(msg string) *AppError {
	return &AppError{Code: CodeNotFound, Message: msg}
}

func NewForbidden(msg string) *AppError {
	return &AppError{Code: CodeForbidden, Message: msg}
}

func NewBadRequest(msg string) *AppError {
	return &AppError{Code: CodeBadRequest, Message: msg}
}

func NewInternal(err error) *AppError {
	return &AppError{Code: CodeInternal, Message: "internal server error", Err: err}
}
