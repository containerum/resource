package routes

import (
	"git.containerum.net/ch/kube-client/pkg/cherry"
	"git.containerum.net/ch/kube-client/pkg/cherry/resource-service"
)

func handleError(err error) (int, *cherry.Err) {
	switch err.(type) {
	case *cherry.Err:
		e := err.(*cherry.Err)
		return e.StatusHTTP, e
	default:
		return rserrors.ErrInternal().StatusHTTP, rserrors.ErrInternal().AddDetailsErr(err)
	}
}

func badRequest(err error) (int, *cherry.Err) {
	return rserrors.ErrValidation().StatusHTTP, rserrors.ErrValidation().AddDetailsErr(err)
}
