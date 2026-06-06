package brerror

import "errors"

var (
	// ErrNotFound indicates that a requested resource could not be found.
	ErrNotFound = errors.New("not found")

	// ErrAlreadyExists indicates that a resource being created already exists.
	ErrAlreadyExists = errors.New("already exists")

	// ErrUnauthenticated indicates that the operation requires authentication
	// but no valid credentials were provided.
	ErrUnauthenticated = errors.New("unauthenticated")

	// ErrInvalidInput indicates that the input failed application-level validation.
	ErrInvalidInput = errors.New("invalid input")

	// ErrResourceInUse indicates the resource is in use and has dependent records.
	ErrResourceInUse = errors.New("resource is still in use and has dependants")
)
