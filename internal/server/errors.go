package server

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrDeleted         = errors.New("deleted")
	ErrVersionConflict = errors.New("version conflict")
	ErrInvalidToken    = errors.New("invalid token")
)
