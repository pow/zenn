package errorpresenter

import "errors"

// GraphQLError は GraphQL レスポンスのエラー表現。
type GraphQLError struct {
	Message    string
	Extensions map[string]interface{}
}

// ClassifyError はエラーを GraphQL レスポンス用に分類する。
func ClassifyError(err error) GraphQLError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return GraphQLError{
			Message:    appErr.Message,
			Extensions: map[string]interface{}{"code": string(appErr.Code)},
		}
	}
	// 未知のエラーは内部情報を漏らさない
	return GraphQLError{
		Message:    "internal server error",
		Extensions: map[string]interface{}{"code": "INTERNAL"},
	}
}
