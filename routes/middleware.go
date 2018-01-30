package routes

import (
	"net/http"

	"git.containerum.net/ch/json-types/errors"
)

func handleError(err error) (int, *errors.Error) {
	switch err.(type) {
	case *errors.Error:
		e := err.(*errors.Error)
		if code := e.Code; code != 0 {
			e.Code = 0
			return code, e
		}
		return http.StatusInternalServerError, e
	default:
		return http.StatusInternalServerError, errors.New(err.Error())
	}
}

func badRequest(err error) (int, *errors.Error) {
	return http.StatusBadRequest, errors.New(err.Error())
}
