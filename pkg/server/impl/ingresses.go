package impl

import (
	"context"

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
	mongo *db.MongoStorage
	log   *cherrylog.LogrusAdapter
}

func NewIngressActionsImpl(mongo *db.MongoStorage) *IngressActionsImpl {
	return &IngressActionsImpl{
		mongo: mongo,
		log:   cherrylog.NewLogrusAdapter(logrus.WithField("component", "ingress_actions")),
	}
}

func (ia *IngressActionsImpl) CreateIngress(ctx context.Context, nsID string, req kubtypes.Ingress) (*ingress.Ingress, error) {
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
		return nil, err
	}

	if server.DetermineServiceType(svc.Service) != service.ServiceExternal {
		return nil, rserrors.ErrServiceNotExternal()
	}

	createdIngress, err := ia.mongo.CreateIngress(ingress.IngressFromKube(nsID, userID, req))
	if err != nil {
		return nil, err
	}
	//TODO
	/*
		if createErr := ia.Kube.CreateIngress(ctx, nsID, req); createErr != nil {
			return createErr
	*/
	return &createdIngress, nil
}

func (ia *IngressActionsImpl) GetIngressesList(ctx context.Context, nsID string) ([]ingress.Ingress, error) {
	userID := httputil.MustGetUserID(ctx)
	ia.log.WithFields(logrus.Fields{
		"user_id":   userID,
		"namespace": nsID,
	}).Info("get user ingresses")

	return ia.mongo.GetIngressList(nsID)
}

func (ia *IngressActionsImpl) GetIngress(ctx context.Context, nsID, ingressName string) (*ingress.Ingress, error) {
	ia.log.Info("get all ingresses")

	resp, err := ia.mongo.GetIngress(nsID, ingressName)

	return &resp, err
}

func (ia *IngressActionsImpl) UpdateIngress(ctx context.Context, nsID string, req kubtypes.Ingress) (*ingress.Ingress, error) {
	userID := httputil.MustGetUserID(ctx)
	ia.log.WithFields(logrus.Fields{
		"user_id": userID,
		"ns_id":   nsID,
		"ingress": req,
	}).Info("update ingress")

	//TODO
	/*
		if delErr := ia.Kube.DeleteIngress(ctx, nsID, ingressName); delErr != nil {
			return delErr
		}
	*/

	ingres, err := ia.mongo.UpdateIngress(ingress.IngressFromKube(nsID, userID, req))
	if err != nil {
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

	//TODO
	/*
		if delErr := ia.Kube.DeleteIngress(ctx, nsID, ingressName); delErr != nil {
			return delErr
		}
	*/

	err := ia.mongo.DeleteIngress(nsID, ingressName)
	if err != nil {
		return err
	}

	return nil
}
