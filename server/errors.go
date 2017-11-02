package server

import (
	"fmt"
)

type OtherServiceError string

type Error string

type PermissionError string

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
