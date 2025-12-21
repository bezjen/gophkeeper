package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorTypes(t *testing.T) {
	t.Run("ErrNotFound", func(t *testing.T) {
		err := ErrNotFound
		assert.Equal(t, "not found", err.Error())
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("ErrDeleted", func(t *testing.T) {
		err := ErrDeleted
		assert.Equal(t, "deleted", err.Error())
		assert.True(t, errors.Is(err, ErrDeleted))
	})

	t.Run("ErrVersionConflict", func(t *testing.T) {
		err := ErrVersionConflict
		assert.Equal(t, "version conflict", err.Error())
		assert.True(t, errors.Is(err, ErrVersionConflict))
	})

	t.Run("ErrInvalidToken", func(t *testing.T) {
		err := ErrInvalidToken
		assert.Equal(t, "invalid token", err.Error())
		assert.True(t, errors.Is(err, ErrInvalidToken))
	})

	t.Run("Проверка цепочки ошибок", func(t *testing.T) {
		wrappedErr := fmt.Errorf("wrapper: %w", ErrNotFound)
		assert.True(t, errors.Is(wrappedErr, ErrNotFound))
	})

	t.Run("Сравнение разных ошибок", func(t *testing.T) {
		assert.False(t, errors.Is(ErrNotFound, ErrDeleted))
		assert.False(t, errors.Is(ErrVersionConflict, ErrInvalidToken))
	})
}

func TestErrorWrapping(t *testing.T) {
	t.Run("Ошибка обернута в fmt.Errorf", func(t *testing.T) {
		err := ErrNotFound
		wrappedErr := fmt.Errorf("context: %w", err)

		assert.True(t, errors.Is(wrappedErr, err))
	})

	t.Run("Множественная вложенность ошибок", func(t *testing.T) {
		innerErr := ErrVersionConflict
		middleErr := fmt.Errorf("middle: %w", innerErr)
		outerErr := fmt.Errorf("outer: %w", middleErr)

		assert.True(t, errors.Is(outerErr, innerErr))
	})
}
