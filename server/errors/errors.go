package errors

import (
	"git.containerum.net/ch/json-types/errors"
)

type (

	// Unhandled error caused by another service.
	OtherServiceError struct {
		e *errors.Error
	}

	// Insufficient access error.
	PermissionError struct {
		e *errors.Error
	}

	// Invalid input error.
	BadInputError struct {
		e *errors.Error
	}
)

func (oe *OtherServiceError) Error() string {
	return oe.e.Error()
}

func (pe *PermissionError) Error() string {
	return pe.e.Error()
}

func (be *BadInputError) Error() string {
	return be.e.Error()
}

var (
	ErrNoSuchResource = errors.New("no such resource")
	ErrAlreadyExists  = errors.New("already exists")
	ErrDenied         = errors.New("permisson denied")
)

func NewOtherServiceError(f string, args ...interface{}) *OtherServiceError {
	return &OtherServiceError{e: errors.Format(f, args...)}
}

func NewPermissionError(f string, args ...interface{}) *PermissionError {
	return &PermissionError{e: errors.Format(f, args...)}
}

func NewBadInputError(f string, args ...interface{}) *BadInputError {
	return &BadInputError{e: errors.Format(f, args...)}
}
