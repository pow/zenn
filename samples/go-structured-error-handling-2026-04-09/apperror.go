package apperror

import (
	"errors"
	"fmt"
	"log/slog"
)

// AppError は 3 層（Code / Message / Detail）で構成される構造化エラーです。
//
//   - Code:    プログラムで判定するための固定識別子（例: "E0001"）
//   - Message: ユーザー向けの固定メッセージ（例: "認証が必要です"）
//   - Detail:  呼び出し元が付与する文脈情報（例: "token expired"）
//   - cause:   内部エラーチェーン（ログ専用、クライアントには返さない）
type AppError struct {
	Code    string
	Message string
	Detail  string
	cause   error
}

// ── 共通エラー定義 ──────────────────────────────────

var (
	ErrUnauthorized = New("E0001", "認証が必要です")
	ErrForbidden    = New("E0002", "権限が不足しています")
	ErrNotFound     = New("E0003", "リソースが見つかりません")
	ErrConflict     = New("E0004", "データが競合しています。再度お試しください")
	ErrInvalidInput = New("E0005", "入力内容が不正です")
	ErrInternal     = New("E0006", "サーバーエラーが発生しました")
)

// ── コンストラクタ ──────────────────────────────────

// New は Code と Message を持つ基本 AppError を生成します。
func New(code, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// ── メソッド ────────────────────────────────────────

// Wrap は cause を内包した新しい AppError を返します。
// 元の Code, Message, Detail はそのまま維持されます。
func (e *AppError) Wrap(cause error) *AppError {
	return &AppError{
		Code:    e.Code,
		Message: e.Message,
		Detail:  e.Detail,
		cause:   cause,
	}
}

// WithDetail は Detail を付与した新しい AppError を返します。
// Message は変更されず codes 定義のまま維持されます。
func (e *AppError) WithDetail(detail string) *AppError {
	return &AppError{
		Code:    e.Code,
		Message: e.Message,
		Detail:  detail,
		cause:   e.cause,
	}
}

// Wrapf は cause のラップと Detail の設定を同時に行います。
func (e *AppError) Wrapf(cause error, format string, args ...any) *AppError {
	return &AppError{
		Code:    e.Code,
		Message: e.Message,
		Detail:  fmt.Sprintf(format, args...),
		cause:   cause,
	}
}

// Cause は内部のエラーチェーンを返します（ログ用途）。
func (e *AppError) Cause() error {
	return e.cause
}

// ── error インターフェース ──────────────────────────

// Error はサーバーサイドログ向けの詳細文字列を返します。
func (e *AppError) Error() string {
	s := fmt.Sprintf("%s: %s", e.Code, e.Message)
	if e.Detail != "" {
		s += ": " + e.Detail
	}
	if e.cause != nil {
		s += ": " + e.cause.Error()
	}
	return s
}

// Unwrap は errors.Is / errors.As によるチェーン走査を可能にします。
func (e *AppError) Unwrap() error {
	return e.cause
}

// Is はエラーコードで一致判定を行います。
// これにより errors.Is(wrappedErr, ErrForbidden) がコードベースで動作します。
func (e *AppError) Is(target error) bool {
	var t *AppError
	if errors.As(target, &t) {
		return e.Code == t.Code
	}
	return false
}

// ── 抽出ヘルパー ───────────────────────────────────

// Extract はエラーチェーンから AppError を取り出します。
// 見つからない場合は ErrInternal を返し、生エラーの漏洩を防ぎます。
func Extract(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return ErrInternal.Wrap(err)
}

// ── 構造化ログ対応 ─────────────────────────────────

// LogValue は slog.LogValuer を実装し、構造化ログに対応します。
func (e *AppError) LogValue() slog.Value {
	attrs := []slog.Attr{
		slog.String("code", e.Code),
		slog.String("message", e.Message),
	}
	if e.Detail != "" {
		attrs = append(attrs, slog.String("detail", e.Detail))
	}
	if e.cause != nil {
		attrs = append(attrs, slog.String("cause", e.cause.Error()))
	}
	return slog.GroupValue(attrs...)
}
