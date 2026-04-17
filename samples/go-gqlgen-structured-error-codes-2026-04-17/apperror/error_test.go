package apperror_test

import (
	"errors"
	"fmt"
	"testing"

	"go-gqlgen-structured-error-codes/apperror"
)

func TestAppError_Error(t *testing.T) {
	t.Run("code and message only", func(t *testing.T) {
		err := apperror.New("E0001", "Authentication required.")
		got := err.Error()
		want := "E0001: Authentication required."
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("with detail", func(t *testing.T) {
		err := apperror.ErrNotFound.WithDetail("user abc-123")
		got := err.Error()
		want := "E0003: Resource not found.: user abc-123"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("with cause", func(t *testing.T) {
		cause := fmt.Errorf("db connection refused")
		err := apperror.ErrInternal.Wrap(cause)
		got := err.Error()
		want := "E0006: Internal server error.: db connection refused"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestAppError_Is(t *testing.T) {
	t.Run("same code matches via errors.Is", func(t *testing.T) {
		err := apperror.ErrNotFound.WithDetail("user xyz")
		if !errors.Is(err, apperror.ErrNotFound) {
			t.Error("expected errors.Is to match ErrNotFound")
		}
	})

	t.Run("different code does not match", func(t *testing.T) {
		err := apperror.ErrNotFound.WithDetail("user xyz")
		if errors.Is(err, apperror.ErrForbidden) {
			t.Error("expected errors.Is NOT to match ErrForbidden")
		}
	})

	t.Run("wrapped AppError matches via errors.Is", func(t *testing.T) {
		inner := apperror.ErrForbidden.Wrap(fmt.Errorf("openfga check failed"))
		outer := fmt.Errorf("use case: %w", inner)
		if !errors.Is(outer, apperror.ErrForbidden) {
			t.Error("expected wrapped AppError to match ErrForbidden")
		}
	})
}

func TestAppError_Wrap(t *testing.T) {
	cause := fmt.Errorf("sql: no rows")
	err := apperror.ErrNotFound.Wrap(cause)

	if err.Code != "E0003" {
		t.Errorf("Code: got %q, want %q", err.Code, "E0003")
	}
	if err.Cause() == nil {
		t.Fatal("Cause should not be nil")
	}
	if err.Cause().Error() != "sql: no rows" {
		t.Errorf("Cause: got %q, want %q", err.Cause().Error(), "sql: no rows")
	}
}

func TestAppError_Wrapf(t *testing.T) {
	cause := fmt.Errorf("sql: no rows")
	err := apperror.ErrNotFound.Wrapf(cause, "user %s", "abc-123")

	if err.Detail != "user abc-123" {
		t.Errorf("Detail: got %q, want %q", err.Detail, "user abc-123")
	}
	if err.Cause() == nil {
		t.Fatal("Cause should not be nil")
	}
}

func TestAppError_WithDetail_does_not_mutate_sentinel(t *testing.T) {
	original := apperror.ErrNotFound
	_ = original.WithDetail("user xyz")

	if original.Detail != "" {
		t.Errorf("Sentinel was mutated: Detail = %q", original.Detail)
	}
}

func TestExtract(t *testing.T) {
	t.Run("extracts AppError from chain", func(t *testing.T) {
		inner := apperror.ErrForbidden.WithDetail("access denied")
		outer := fmt.Errorf("resolver: %w", inner)

		got := apperror.Extract(outer)
		if got.Code != "E0002" {
			t.Errorf("Code: got %q, want %q", got.Code, "E0002")
		}
		if got.Detail != "access denied" {
			t.Errorf("Detail: got %q, want %q", got.Detail, "access denied")
		}
	})

	t.Run("returns ErrInternal for unknown error", func(t *testing.T) {
		err := fmt.Errorf("unexpected: nil pointer")
		got := apperror.Extract(err)
		if got.Code != "E0006" {
			t.Errorf("Code: got %q, want %q", got.Code, "E0006")
		}
	})

	t.Run("returns ErrInternal for nil-containing error", func(t *testing.T) {
		err := fmt.Errorf("some error")
		got := apperror.Extract(err)
		if !errors.Is(got, apperror.ErrInternal) {
			t.Error("expected ErrInternal for non-AppError")
		}
	})
}
