package service

import "errors"

var (
	// ErrNotFound indicates a resource was not found
	ErrNotFound = errors.New("resource not found")

	// ErrForbidden indicates the user is not allowed to perform this action
	ErrForbidden = errors.New("forbidden")

	// ErrConflict indicates a conflict with the current state
	ErrConflict = errors.New("conflict")

	// ErrInvalidState indicates the resource is in an invalid state for this operation
	ErrInvalidState = errors.New("invalid state for this operation")

	// ErrAlreadyExists indicates the resource already exists
	ErrAlreadyExists = errors.New("resource already exists")

	// ErrSelfAction indicates the user tried to perform an action on themselves
	ErrSelfAction = errors.New("cannot perform this action on yourself")
)
