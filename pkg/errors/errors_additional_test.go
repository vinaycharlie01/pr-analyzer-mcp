package errors_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	apperrors "github.com/vinaycharlie01/pr-analyzer-mcp/pkg/errors"
)

func TestIsUnauthorized(t *testing.T) {
	auth := apperrors.New(apperrors.ErrCodeUnauthorized, "unauthorized")
	other := apperrors.New(apperrors.ErrCodeInternal, "internal")

	assert.True(t, apperrors.IsUnauthorized(auth))
	assert.False(t, apperrors.IsUnauthorized(other))
}

func TestAllErrorCodes(t *testing.T) {
	codes := []apperrors.ErrorCode{
		apperrors.ErrCodeNotFound,
		apperrors.ErrCodeUnauthorized,
		apperrors.ErrCodeBadRequest,
		apperrors.ErrCodeInternal,
		apperrors.ErrCodeTimeout,
		apperrors.ErrCodeRateLimit,
		apperrors.ErrCodeAnalysis,
		apperrors.ErrCodeConfiguration,
	}

	for _, code := range codes {
		err := apperrors.New(code, "test message")
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), string(code))
		assert.Contains(t, err.Error(), "test message")
	}
}

func TestAppError_Unwrap_Nil(t *testing.T) {
	err := apperrors.New(apperrors.ErrCodeNotFound, "not found")
	assert.Nil(t, errors.Unwrap(err))
}

func TestIsNotFound_WithStdErr(t *testing.T) {
	stdErr := errors.New("standard error")
	assert.False(t, apperrors.IsNotFound(stdErr))
	assert.False(t, apperrors.IsUnauthorized(stdErr))
	assert.False(t, apperrors.IsRateLimit(stdErr))
}

func TestIsConfiguration(t *testing.T) {
	configErr := apperrors.New(apperrors.ErrCodeConfiguration, "not configured")
	other := apperrors.New(apperrors.ErrCodeInternal, "internal")

	assert.True(t, apperrors.IsConfiguration(configErr))
	assert.False(t, apperrors.IsConfiguration(other))
	assert.False(t, apperrors.IsConfiguration(errors.New("std error")))
}

func TestWrap_ErrorIs(t *testing.T) {
	sentinel := errors.New("sentinel")
	wrapped := apperrors.Wrap(apperrors.ErrCodeInternal, "wrap 1", sentinel)
	doubleWrapped := apperrors.Wrap(apperrors.ErrCodeInternal, "wrap 2", wrapped)

	assert.ErrorIs(t, doubleWrapped, sentinel)
}
