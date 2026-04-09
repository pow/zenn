package apperror

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// ErrorResponse は API クライアントに返すエラーレスポンスの形式です。
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// codeToStatus はエラーコードを HTTP ステータスコードに変換します。
func codeToStatus(code string) int {
	switch code {
	case ErrUnauthorized.Code:
		return http.StatusUnauthorized
	case ErrForbidden.Code:
		return http.StatusForbidden
	case ErrNotFound.Code:
		return http.StatusNotFound
	case ErrConflict.Code:
		return http.StatusConflict
	case ErrInvalidInput.Code:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// WriteError は AppError を JSON レスポンスとして書き込みます。
// 未知のエラーは自動的に ErrInternal に変換され、生エラーの漏洩を防ぎます。
func WriteError(w http.ResponseWriter, err error) {
	appErr := Extract(err)

	// サーバーサイドには完全なエラー情報をログ出力
	slog.Error("request error", slog.Any("error", appErr))

	// クライアントには安全な情報のみ返す
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(codeToStatus(appErr.Code))
	json.NewEncoder(w).Encode(ErrorResponse{
		Code:    appErr.Code,
		Message: appErr.Message,
		Detail:  appErr.Detail,
	})
}
