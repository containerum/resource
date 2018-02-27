package clients

import (
	"context"
	"fmt"
	"net/url"

	"git.containerum.net/ch/json-types/errors"
	"git.containerum.net/ch/kube-client/pkg/cherry"
	"git.containerum.net/ch/kube-client/pkg/cherry/adaptors/cherrylog"
	"git.containerum.net/ch/kube-client/pkg/cherry/resource-service"
	kubtypes "git.containerum.net/ch/kube-client/pkg/model"
	"git.containerum.net/ch/utils"
	"github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
)

// Kube is an interface to kube-api service
type Kube interface {
	CreateNamespace(ctx context.Context, ns kubtypes.Namespace) error
	SetNamespaceQuota(ctx context.Context, ns kubtypes.Namespace) error
	DeleteNamespace(ctx context.Context, label string) error
}

type kube struct {
	client *resty.Client
	log    *cherrylog.LogrusAdapter
}

// NewKubeHTTP creates http client to kube-api service.
func NewKubeHTTP(u *url.URL) Kube {
	log := logrus.WithField("component", "kube_client")
	client := resty.New().
		SetHostURL(u.String()).
		SetLogger(log.WriterLevel(logrus.DebugLevel)).
		SetDebug(true).
		SetError(cherry.Err{})
	client.JSONMarshal = jsoniter.Marshal
	client.JSONUnmarshal = jsoniter.Unmarshal
	return kube{
		client: client,
		log:    cherrylog.NewLogrusAdapter(log),
	}
}

func (kub kube) CreateNamespace(ctx context.Context, ns kubtypes.Namespace) error {
	kub.log.WithFields(logrus.Fields{
		"cpu":    ns.Resources.Hard.CPU,
		"memory": ns.Resources.Hard.Memory,
		"name":   ns.Label,
		"access": ns.Access,
	}).Infoln("create namespace")

	resp, err := kub.client.R().
		SetBody(ns).
		SetContext(ctx).
		SetHeaders(utils.RequestHeadersMap(ctx)).
		Post("/namespaces")
	if err != nil {
		return rserrors.ErrOther().Log(err, kub.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}
	return nil
}

func (kub kube) DeleteNamespace(ctx context.Context, label string) error {
	kub.log.WithField("label", label).Infoln("delete namespace")

	resp, err := kub.client.R().
		SetContext(ctx).
		SetHeaders(utils.RequestHeadersMap(ctx)).
		Delete("/namespaces/" + url.PathEscape(label))
	if err != nil {
		return rserrors.ErrOther().Log(err, kub.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}
	return nil
}

func (kub kube) SetNamespaceQuota(ctx context.Context, ns kubtypes.Namespace) error {
	kub.log.WithFields(logrus.Fields{
		"cpu":    ns.Resources.Hard.CPU,
		"memory": ns.Resources.Hard.Memory,
		"label":  ns.Label,
	}).Infoln("set namespace quota")

	resp, err := kub.client.R().
		SetBody(ns).
		SetContext(ctx).
		SetHeaders(utils.RequestHeadersMap(ctx)).
		Put("/namespaces/" + url.PathEscape(ns.Label))
	if err != nil {
		return rserrors.ErrOther().Log(err, kub.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*errors.Error)
	}
	return nil
}

func (kub kube) String() string {
	return fmt.Sprintf("kube api http client: url=%v", kub.client.HostURL)
}

type kubeDummy struct {
	log *logrus.Entry
}

// NewDummyKube creates a dummy client to kube-api service. It does nothing but logs actions.
func NewDummyKube() Kube {
	return kubeDummy{log: logrus.WithField("component", "kube_stub")}
}

func (kub kubeDummy) CreateNamespace(_ context.Context, ns kubtypes.Namespace) error {
	kub.log.WithFields(logrus.Fields{
		"cpu":    ns.Resources.Hard.CPU,
		"memory": ns.Resources.Hard.Memory,
		"name":   ns.Label,
		"access": ns.Access,
	}).Infoln("create namespace")
	return nil
}

func (kub kubeDummy) DeleteNamespace(_ context.Context, label string) error {
	kub.log.WithField("label", label).Infoln("delete namespace")
	return nil
}

func (kub kubeDummy) SetNamespaceQuota(_ context.Context, ns kubtypes.Namespace) error {
	kub.log.WithFields(logrus.Fields{
		"cpu":    ns.Resources.Hard.CPU,
		"memory": ns.Resources.Hard.Memory,
		"label":  ns.Label,
	}).Infoln("set namespace quota")
	return nil
}

func (kubeDummy) String() string {
	return "kube api dummy"
}
