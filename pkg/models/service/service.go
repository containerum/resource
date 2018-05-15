package service

import (
	"github.com/containerum/kube-client/pkg/model"
	"github.com/google/uuid"
)

type Service struct {
	model.Service
	Owner       string `json:"owner"`
	ID          string `json:"_id"`
	Deleted     bool   `json:"deleted"`
	NamespaceID string `json:"namespace_id"`
}

func ServiceFromKube(nsID, owner string, service model.Service) Service {
	return Service{
		Service:     service,
		Owner:       owner,
		NamespaceID: nsID,
		ID:          uuid.New().String(),
	}
}
