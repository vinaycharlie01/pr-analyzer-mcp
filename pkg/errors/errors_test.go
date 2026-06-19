package errors_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	apperrors "github.com/vinaycharlie01/pr-analyzer-mcp/pkg/errors"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *apperrors.AppError
		contains string
	}{
		{
			name:     "without wrapped error",
			err:      apperrors.New(apperrors.ErrCodeNotFound, "resource not found"),
			contains: "NOT_FOUND",
		},
		{
			name:     "with wrapped error",
			err:      apperrors.Wrap(apperrors.ErrCodeInternal, "something failed", errors.New("underlying")),
			contains: "underlying",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Contains(t, tc.err.Error(), tc.contains)
		})
	}
}

func TestIsNotFound(t *testing.T) {
	notFound := apperrors.New(apperrors.ErrCodeNotFound, "not found")
	wrapped := fmt.Errorf("wrapping: %w", notFound)
	other := apperrors.New(apperrors.ErrCodeInternal, "internal")

	assert.True(t, apperrors.IsNotFound(notFound))
	assert.False(t, apperrors.IsNotFound(other))
	_ = wrapped
}

func TestIsRateLimit(t *testing.T) {
	rl := apperrors.New(apperrors.ErrCodeRateLimit, "rate limited")
	assert.True(t, apperrors.IsRateLimit(rl))
	assert.False(t, apperrors.IsRateLimit(apperrors.New(apperrors.ErrCodeNotFound, "not found")))
}

func TestUnwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := apperrors.Wrap(apperrors.ErrCodeInternal, "operation failed", cause)
	assert.ErrorIs(t, err, cause)
}
