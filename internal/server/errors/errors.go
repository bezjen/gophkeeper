// Package errors defines common error types used throughout the GophKeeper application.
package errors

import "errors"

var (
	// ErrNotFound indicates that a requested resource was not found.
	ErrNotFound = errors.New("not found")

	// ErrDeleted indicates that a requested resource has been deleted.
	ErrDeleted = errors.New("deleted")

	// ErrVersionConflict indicates a version conflict during data update operations.
	ErrVersionConflict = errors.New("version conflict")

	// ErrInvalidToken indicates that an authentication token is invalid or expired.
	ErrInvalidToken = errors.New("invalid token")
)
