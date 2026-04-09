package apperror_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	apperror "go-structured-error-handling"
)

// ── AppError 基本動作 ──────────────────────────────

func TestNew_creates_error_with_code_and_message(t *testing.T) {
	err := apperror.New("E0100", "ユーザーが見つかりません")

	if err.Code != "E0100" {
		t.Errorf("Code = %q, want %q", err.Code, "E0100")
	}
	if err.Message != "ユーザーが見つかりません" {
		t.Errorf("Message = %q, want %q", err.Message, "ユーザーが見つかりません")
	}
}

func TestWrap_preserves_code_and_message(t *testing.T) {
	cause := errors.New("connection refused")
	wrapped := apperror.ErrInternal.Wrap(cause)

	if wrapped.Code != "E0006" {
		t.Errorf("Code = %q, want %q", wrapped.Code, "E0006")
	}
	if wrapped.Message != "サーバーエラーが発生しました" {
		t.Errorf("Message = %q, want %q", wrapped.Message, "サーバーエラーが発生しました")
	}
	if wrapped.Cause() != cause {
		t.Error("Cause should be the original error")
	}
}

func TestWithDetail_adds_context_without_changing_message(t *testing.T) {
	detailed := apperror.ErrForbidden.WithDetail("admin role required")

	if detailed.Message != "権限が不足しています" {
		t.Errorf("Message should not change, got %q", detailed.Message)
	}
	if detailed.Detail != "admin role required" {
		t.Errorf("Detail = %q, want %q", detailed.Detail, "admin role required")
	}
}

func TestWrapf_sets_detail_and_cause(t *testing.T) {
	cause := errors.New("token expired")
	err := apperror.ErrUnauthorized.Wrapf(cause, "session %s is invalid", "abc123")

	if err.Detail != "session abc123 is invalid" {
		t.Errorf("Detail = %q, want %q", err.Detail, "session abc123 is invalid")
	}
	if err.Cause() != cause {
		t.Error("Cause should be the original error")
	}
}

// ── Error() 文字列フォーマット ─────────────────────

func TestError_includes_all_layers(t *testing.T) {
	cause := errors.New("disk full")
	err := apperror.ErrInternal.Wrapf(cause, "failed to save file")

	want := "E0006: サーバーエラーが発生しました: failed to save file: disk full"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}

func TestError_omits_empty_detail_and_cause(t *testing.T) {
	want := "E0003: リソースが見つかりません"
	if apperror.ErrNotFound.Error() != want {
		t.Errorf("Error() = %q, want %q", apperror.ErrNotFound.Error(), want)
	}
}

// ── errors.Is によるコードベースマッチング ─────────

func TestIs_matches_by_code_after_wrap(t *testing.T) {
	cause := errors.New("record not found")
	wrapped := apperror.ErrNotFound.Wrap(cause)

	if !errors.Is(wrapped, apperror.ErrNotFound) {
		t.Error("errors.Is should match by code even after Wrap")
	}
}

func TestIs_matches_through_fmt_errorf(t *testing.T) {
	original := apperror.ErrForbidden.WithDetail("access denied")
	wrapped := fmt.Errorf("usecase failed: %w", original)

	if !errors.Is(wrapped, apperror.ErrForbidden) {
		t.Error("errors.Is should match through fmt.Errorf wrapping")
	}
}

func TestIs_does_not_match_different_code(t *testing.T) {
	if errors.Is(apperror.ErrNotFound, apperror.ErrForbidden) {
		t.Error("Different codes should not match")
	}
}

// ── Extract 防御的抽出 ─────────────────────────────

func TestExtract_returns_apperror_from_chain(t *testing.T) {
	original := apperror.ErrForbidden.WithDetail("no permission")
	wrapped := fmt.Errorf("handler: %w", original)
	extracted := apperror.Extract(wrapped)

	if extracted.Code != "E0002" {
		t.Errorf("Code = %q, want %q", extracted.Code, "E0002")
	}
	if extracted.Detail != "no permission" {
		t.Errorf("Detail = %q, want %q", extracted.Detail, "no permission")
	}
}

func TestExtract_converts_unknown_error_to_internal(t *testing.T) {
	rawErr := errors.New("unexpected panic")
	extracted := apperror.Extract(rawErr)

	if extracted.Code != apperror.ErrInternal.Code {
		t.Errorf("Code = %q, want %q", extracted.Code, apperror.ErrInternal.Code)
	}
	if extracted.Cause() != rawErr {
		t.Error("Cause should preserve the original unknown error")
	}
}

// ── HTTP レスポンス変換 ────────────────────────────

func TestWriteError_NotFound_returns_404(t *testing.T) {
	w := httptest.NewRecorder()
	err := apperror.ErrNotFound.WithDetail("user 42 not found")

	apperror.WriteError(w, err)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var resp apperror.ErrorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != "E0003" {
		t.Errorf("response code = %q, want %q", resp.Code, "E0003")
	}
	if resp.Detail != "user 42 not found" {
		t.Errorf("response detail = %q, want %q", resp.Detail, "user 42 not found")
	}
}

func TestWriteError_unknown_error_returns_500(t *testing.T) {
	w := httptest.NewRecorder()
	rawErr := errors.New("nil pointer dereference")

	apperror.WriteError(w, rawErr)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var resp apperror.ErrorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != "E0006" {
		t.Errorf("response code = %q, want %q", resp.Code, "E0006")
	}
}

func TestWriteError_Forbidden_returns_403(t *testing.T) {
	w := httptest.NewRecorder()
	err := apperror.ErrForbidden.WithDetail("admin only")

	apperror.WriteError(w, err)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}
