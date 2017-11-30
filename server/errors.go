package server

import (
	"fmt"
)

type (
	OtherServiceError string
	Error             string
	PermissionError   string
	BadInputError     string
)

var (
	ErrNoSuchResource = Error("no such resource")
	ErrAlreadyExists = Error("already exists")
	ErrDenied = PermissionError("denied")
)

func newOtherServiceError(f string, args ...interface{}) OtherServiceError {
	return OtherServiceError(fmt.Sprintf(f, args...))
}

func (e OtherServiceError) Error() string {
	return string(e)
}

func newError(f string, args ...interface{}) Error {
	return Error(fmt.Sprintf(f, args...))
}

func (e Error) Error() string {
	return string(e)
}

func newPermissionError(f string, args ...interface{}) PermissionError {
	return PermissionError(fmt.Sprintf(f, args...))
}

func (e PermissionError) Error() string {
	return string(e)
}

func newBadInputError(f string, args ...interface{}) BadInputError {
	return BadInputError(fmt.Sprintf(f, args...))
}

func (e BadInputError) Error() string {
	return string(e)
}

//func newNoObjectError(f string, args ...interface{}) NoObjectError {
//	return NoObjectError(fmt.Sprintf(f, args...))
//}
//
////func (e NoObjectError) Error() string {
//	return string(e)
//}
