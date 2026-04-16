package errorpresenter

import (
	"fmt"
	"testing"
)

func TestAppErrorNotFound(t *testing.T) {
	err := NewNotFound("user not found")
	if err.Code != CodeNotFound {
		t.Errorf("expected code %s, got %s", CodeNotFound, err.Code)
	}
	if err.Message != "user not found" {
		t.Errorf("expected message 'user not found', got '%s'", err.Message)
	}
}

func TestAppErrorWithWrappedError(t *testing.T) {
	cause := fmt.Errorf("connection refused")
	err := NewInternal(cause)

	if err.Unwrap() != cause {
		t.Error("Unwrap should return the wrapped error")
	}
	if err.Code != CodeInternal {
		t.Errorf("expected code %s, got %s", CodeInternal, err.Code)
	}
}

func TestAppErrorErrorString(t *testing.T) {
	t.Run("without wrapped error", func(t *testing.T) {
		err := NewNotFound("item not found")
		expected := "NOT_FOUND: item not found"
		if err.Error() != expected {
			t.Errorf("expected '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("with wrapped error", func(t *testing.T) {
		cause := fmt.Errorf("db error")
		err := NewInternal(cause)
		expected := "INTERNAL: internal server error: db error"
		if err.Error() != expected {
			t.Errorf("expected '%s', got '%s'", expected, err.Error())
		}
	})
}

func TestClassifyErrorWithAppError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode string
		wantMsg  string
	}{
		{
			name:     "NotFound returns NOT_FOUND code",
			err:      NewNotFound("user not found"),
			wantCode: "NOT_FOUND",
			wantMsg:  "user not found",
		},
		{
			name:     "Forbidden returns FORBIDDEN code",
			err:      NewForbidden("access denied"),
			wantCode: "FORBIDDEN",
			wantMsg:  "access denied",
		},
		{
			name:     "BadRequest returns BAD_REQUEST code",
			err:      NewBadRequest("invalid input"),
			wantCode: "BAD_REQUEST",
			wantMsg:  "invalid input",
		},
		{
			name:     "Internal returns INTERNAL code",
			err:      NewInternal(fmt.Errorf("db down")),
			wantCode: "INTERNAL",
			wantMsg:  "internal server error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.err)
			if result.Extensions["code"] != tt.wantCode {
				t.Errorf("expected code %s, got %s", tt.wantCode, result.Extensions["code"])
			}
			if result.Message != tt.wantMsg {
				t.Errorf("expected message '%s', got '%s'", tt.wantMsg, result.Message)
			}
		})
	}
}

func TestClassifyErrorWithWrappedAppError(t *testing.T) {
	original := NewNotFound("item not found")
	wrapped := fmt.Errorf("repository: %w", original)

	result := ClassifyError(wrapped)

	if result.Extensions["code"] != "NOT_FOUND" {
		t.Errorf("expected NOT_FOUND, got %s", result.Extensions["code"])
	}
	if result.Message != "item not found" {
		t.Errorf("expected 'item not found', got '%s'", result.Message)
	}
}

func TestClassifyErrorWithUnknownError(t *testing.T) {
	err := fmt.Errorf("unexpected panic")
	result := ClassifyError(err)

	if result.Extensions["code"] != "INTERNAL" {
		t.Errorf("expected INTERNAL, got %s", result.Extensions["code"])
	}
	if result.Message != "internal server error" {
		t.Errorf("expected 'internal server error', got '%s'", result.Message)
	}
}
