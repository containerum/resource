package impl

import (
	"context"

	"git.containerum.net/ch/resource-service/pkg/clients"
	"git.containerum.net/ch/resource-service/pkg/db"
	"git.containerum.net/ch/resource-service/pkg/models/ingress"
	"git.containerum.net/ch/resource-service/pkg/rsErrors"
	"git.containerum.net/ch/resource-service/pkg/server"
	"git.containerum.net/ch/resource-service/pkg/util/coblog"
	"github.com/containerum/cherry/adaptors/cherrylog"
	kubtypes "github.com/containerum/kube-client/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/idna"
)

type IngressActionsImpl struct {
	kube   clients.Kube
	mongo  *db.MongoStorage
	log    *cherrylog.LogrusAdapter
	suffix string
}

func NewIngressActionsImpl(mongo *db.MongoStorage, kube *clients.Kube, ingressSuffix string) *IngressActionsImpl {
	return &IngressActionsImpl{
		kube:   *kube,
		mongo:  mongo,
		log:    cherrylog.NewLogrusAdapter(logrus.WithField("component", "ingress_actions")),
		suffix: ingressSuffix,
	}
}

func (ia *IngressActionsImpl) GetIngressesList(ctx context.Context, nsID string) (*ingress.IngressesResponse, error) {
	userID := httputil.MustGetUserID(ctx)
	ia.log.WithFields(logrus.Fields{
		"user_id":   userID,
		"namespace": nsID,
	}).Info("get user ingresses")

	ingresses, err := ia.mongo.GetIngressList(nsID)
	if err != nil {
		return nil, err
	}

	return &ingress.IngressesResponse{Ingresses: ingresses}, nil
}

func (ia *IngressActionsImpl) GetSelectedIngressesList(ctx context.Context, namespaces []string) (*ingress.IngressesResponse, error) {
	userID := httputil.MustGetUserID(ctx)
	ia.log.WithFields(logrus.Fields{
		"user_id":    userID,
		"namespaces": namespaces,
	}).Info("get selected ingresses")

	ingresses, err := ia.mongo.GetSelectedIngresses(namespaces)
	if err != nil {
		return nil, err
	}

	return &ingress.IngressesResponse{Ingresses: ingresses}, nil
}

func (ia *IngressActionsImpl) GetIngress(ctx context.Context, nsID, ingressName string) (*ingress.ResourceIngress, error) {
	ia.log.Info("get all ingresses")

	resp, err := ia.mongo.GetIngress(nsID, ingressName)

	return &resp, err
}

func (ia *IngressActionsImpl) CreateIngress(ctx context.Context, nsID string, req kubtypes.Ingress) (*ingress.ResourceIngress, error) {
	userID := httputil.MustGetUserID(ctx)
	ia.log.WithFields(logrus.Fields{
		"user_id": userID,
		"ns_id":   nsID,
	}).Info("create ingress")
	coblog.Std.Struct(req)

	//Convert host to dns-label, validate it and append suffix
	var err error
	req.Rules[0].Host, err = idna.Lookup.ToASCII(req.Rules[0].Host)
	if err != nil {
		return nil, rserrors.ErrValidation().AddDetailsErr(err)
	}

	req.Rules[0].Host = req.Rules[0].Host + ia.suffix

	if req.Rules[0].Path[0].Path == "" {
		req.Rules[0].Path[0].Path = "/"
	}

	svc, err := ia.mongo.GetService(nsID, req.Rules[0].Path[0].ServiceName)
	if err != nil {
		ia.log.Error(err)
		return nil, rserrors.ErrResourceNotExists().AddDetailF("service '%v' not exists", req.Rules[0].Path[0].ServiceName)
	}

	req.Rules[0].Path, err = server.IngressPaths(svc.Service, req.Rules[0].Path[0].Path, req.Rules[0].Path[0].ServicePort)
	if err != nil {
		return nil, err
	}

	createdIngress, err := ia.mongo.CreateIngress(ingress.FromKube(nsID, userID, req))
	if err != nil {
		return nil, err
	}

	if err := ia.kube.CreateIngress(ctx, nsID, req); err != nil {
		ia.log.Debug("Kube-API error! Deleting ingress from DB.")
		if err := ia.mongo.DeleteIngress(nsID, req.Name); err != nil {
			return nil, err
		}
		return nil, err
	}

	return &createdIngress, nil
}

func (ia *IngressActionsImpl) UpdateIngress(ctx context.Context, nsID string, req kubtypes.Ingress) (*ingress.ResourceIngress, error) {
	userID := httputil.MustGetUserID(ctx)
	ia.log.WithFields(logrus.Fields{
		"user_id": userID,
		"ns_id":   nsID,
		"ingress": req,
	}).Info("update ingress")

	oldIngress, err := ia.mongo.GetIngress(nsID, req.Name)
	if err != nil {
		return nil, err
	}

	req.Rules[0].Host = req.Rules[0].Host + ia.suffix
	req.Name = oldIngress.Name

	if req.Rules[0].Path[0].Path == "" {
		req.Rules[0].Path[0].Path = "/"
	}

	svc, err := ia.mongo.GetService(nsID, req.Rules[0].Path[0].ServiceName)
	if err != nil {
		ia.log.Error(err)
		return nil, rserrors.ErrResourceNotExists().AddDetails("service '%v' not exists", req.Rules[0].Path[0].ServiceName)
	}

	req.Rules[0].Path, err = server.IngressPaths(svc.Service, req.Rules[0].Path[0].Path, req.Rules[0].Path[0].ServicePort)
	if err != nil {
		return nil, err
	}

	ingres, err := ia.mongo.UpdateIngress(ingress.FromKube(nsID, userID, req))
	if err != nil {
		return nil, err
	}

	if err := ia.kube.UpdateIngress(ctx, nsID, req); err != nil {
		ia.log.Debug("Kube-API error! Reverting changes.")
		if _, err := ia.mongo.UpdateIngress(oldIngress); err != nil {
			return nil, err
		}
		return nil, err
	}

	return &ingres, nil
}

func (ia *IngressActionsImpl) DeleteIngress(ctx context.Context, nsID, ingressName string) error {
	userID := httputil.MustGetUserID(ctx)
	ia.log.WithFields(logrus.Fields{
		"user_id": userID,
		"ns_id":   nsID,
		"domain":  ingressName,
	}).Info("delete ingress")

	if err := ia.mongo.DeleteIngress(nsID, ingressName); err != nil {
		return err
	}

	if err := ia.kube.DeleteIngress(ctx, nsID, ingressName); err != nil {
		ia.log.Debug("Kube-API error! Reverting changes.")
		if err := ia.mongo.RestoreIngress(nsID, ingressName); err != nil {
			return err
		}
		return err
	}

	return nil
}

func (ia *IngressActionsImpl) DeleteAllIngresses(ctx context.Context, nsID string) error {
	ia.log.WithFields(logrus.Fields{
		"ns_id": nsID,
	}).Info("delete all ingresses")

	if err := ia.mongo.DeleteAllIngressesInNamespace(nsID); err != nil {
		return err
	}

	return nil
}
