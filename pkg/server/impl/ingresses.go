package impl

import (
	"context"

	"git.containerum.net/ch/resource-service/pkg/clients"
	"git.containerum.net/ch/resource-service/pkg/db"
	"git.containerum.net/ch/resource-service/pkg/models/ingress"
	"git.containerum.net/ch/resource-service/pkg/models/service"
	"git.containerum.net/ch/resource-service/pkg/rsErrors"
	"git.containerum.net/ch/resource-service/pkg/server"
	"github.com/containerum/cherry/adaptors/cherrylog"
	kubtypes "github.com/containerum/kube-client/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/idna"
)

const (
	ingressHostSuffix = ".hub.containerum.io"
)

type IngressActionsImpl struct {
	kube  clients.Kube
	mongo *db.MongoStorage
	log   *cherrylog.LogrusAdapter
}

func NewIngressActionsImpl(mongo *db.MongoStorage, kube *clients.Kube) *IngressActionsImpl {
	return &IngressActionsImpl{
		kube:  *kube,
		mongo: mongo,
		log:   cherrylog.NewLogrusAdapter(logrus.WithField("component", "ingress_actions")),
	}
}

func (ia *IngressActionsImpl) GetIngressesList(ctx context.Context, nsID string) (ingress.IngressList, error) {
	userID := httputil.MustGetUserID(ctx)
	ia.log.WithFields(logrus.Fields{
		"user_id":   userID,
		"namespace": nsID,
	}).Info("get user ingresses")

	return ia.mongo.GetIngressList(nsID)
}

func (ia *IngressActionsImpl) GetIngress(ctx context.Context, nsID, ingressName string) (*ingress.IngressResource, error) {
	ia.log.Info("get all ingresses")

	resp, err := ia.mongo.GetIngress(nsID, ingressName)

	return &resp, err
}

func (ia *IngressActionsImpl) CreateIngress(ctx context.Context, nsID string, req kubtypes.Ingress) (*ingress.IngressResource, error) {
	userID := httputil.MustGetUserID(ctx)
	ia.log.WithFields(logrus.Fields{
		"user_id": userID,
		"ns_id":   nsID,
	}).Infof("create ingress %#v", req)

	//Convert host to dns-label, validate it and append ".hub.containerum.io"
	var err error
	req.Rules[0].Host, err = idna.Lookup.ToASCII(req.Rules[0].Host)
	if err != nil {
		return nil, rserrors.ErrValidation().AddDetailsErr(err)
	}

	req.Rules[0].Host = req.Rules[0].Host + ingressHostSuffix
	req.Name = req.Rules[0].Host

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

	if server.DetermineServiceType(svc.Service) != service.ServiceExternal {
		return nil, rserrors.ErrServiceNotExternal()
	}

	createdIngress, err := ia.mongo.CreateIngress(ingress.IngressFromKube(nsID, userID, req))
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

func (ia *IngressActionsImpl) UpdateIngress(ctx context.Context, nsID string, req kubtypes.Ingress) (*ingress.IngressResource, error) {
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

	req.Rules[0].Path[0].ServiceName = oldIngress.Rules[0].Path[0].ServiceName

	req.Rules[0].Host = req.Rules[0].Host + ingressHostSuffix
	req.Name = req.Rules[0].Host

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

	ingres, err := ia.mongo.UpdateIngress(ingress.IngressFromKube(nsID, userID, req))
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
