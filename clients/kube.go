package clients

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"

	"git.containerum.net/ch/json-types/errors"
	rstypes "git.containerum.net/ch/json-types/resource-service"
	"github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
)

// Kube is an interface to kube-api service
type Kube interface {
	CreateNamespace(ctx context.Context, label string, ns rstypes.NamespaceWithPermission) error
	SetNamespaceQuota(ctx context.Context, label string, ns rstypes.NamespaceWithPermission) error
	DeleteNamespace(ctx context.Context, ns rstypes.Namespace) error
}

type kube struct {
	client *resty.Client
	log    *logrus.Entry
}

const namespaceHeader = "x-user-namespace"

// NewKubeHTTP creates http client to kube-api service.
func NewKubeHTTP(u *url.URL) Kube {
	log := logrus.WithField("component", "kube_client")
	client := resty.New().
		SetHostURL(u.String()).
		SetLogger(log.WriterLevel(logrus.DebugLevel)).
		SetDebug(true).
		SetError(errors.Error{})
	client.JSONMarshal = jsoniter.Marshal
	client.JSONUnmarshal = jsoniter.Unmarshal
	return kube{
		client: client,
		log:    log,
	}
}

type kubNamespaceHeaderElement struct {
	ID     string                   `json:"id,omitempty"`
	Label  string                   `json:"label,omitempty"`
	Access rstypes.PermissionStatus `json:"access,omitempty"`
}

func (kub kube) createNamespaceHeaderValue(e kubNamespaceHeaderElement) string {
	xUserNamespaceBytes, _ := kub.client.JSONMarshal([]kubNamespaceHeaderElement{e})
	return base64.StdEncoding.EncodeToString(xUserNamespaceBytes)
}

func (kub kube) CreateNamespace(ctx context.Context, label string, ns rstypes.NamespaceWithPermission) error {
	kub.log.WithFields(logrus.Fields{
		"name":   ns.ID,
		"cpu":    ns.CPU,
		"memory": ns.RAM,
		"label":  label,
		"access": ns.AccessLevel,
	}).Infoln("create namespace")

	resp, err := kub.client.R().SetBody(map[string]interface{}{
		"kind":       "Namespace",
		"apiVersion": "v1",
		"metadata": map[string]interface{}{
			"name": ns.ID,
		},
	}).
		SetQueryParam("cpu", fmt.Sprint(ns.CPU)).
		SetQueryParam("memory", fmt.Sprint(ns.RAM)).
		SetHeader(namespaceHeader, kub.createNamespaceHeaderValue(kubNamespaceHeaderElement{
			ID:     ns.ID,
			Label:  label,
			Access: ns.AccessLevel,
		})).
		SetContext(ctx).
		Post("api/v1/namespaces")
	if err != nil {
		return err
	}
	if resp.Error() != nil {
		return resp.Error().(*errors.Error)
	}
	return nil
}

func (kub kube) DeleteNamespace(ctx context.Context, ns rstypes.Namespace) error {
	kub.log.WithField("name", ns.ID).Infoln("delete namespace")

	resp, err := kub.client.R().
		SetHeader(namespaceHeader, kub.createNamespaceHeaderValue(kubNamespaceHeaderElement{
			ID:     ns.ID,
			Label:  "unknown", // not important though
			Access: "owner",
		})).
		SetContext(ctx).
		Delete("api/v1/namespaces/" + url.PathEscape(ns.ID))
	if err != nil {
		return err
	}
	if resp.Error() != nil {
		return resp.Error().(*errors.Error)
	}
	return nil
}

func (kub kube) SetNamespaceQuota(ctx context.Context, label string, ns rstypes.NamespaceWithPermission) (err error) {
	// TODO: update cpu and memory also

	kub.log.WithFields(logrus.Fields{
		"name":   ns.ID,
		"cpu":    ns.CPU,
		"memory": ns.RAM,
		"label":  label,
		"access": ns.AccessLevel,
	}).Infoln("set namespace quota")

	resp, err := kub.client.R().
		SetHeader(namespaceHeader, kub.createNamespaceHeaderValue(kubNamespaceHeaderElement{
			ID:     ns.ID,
			Access: "owner",
		})).
		SetContext(ctx).
		Put("api/v1/namespaces/" + url.PathEscape(ns.ID) + "/resourcequotas/quota")
	if err != nil {
		return
	}
	if resp.Error() != nil {
		err = resp.Error().(*errors.Error)
	}
	return
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

func (kub kubeDummy) CreateNamespace(_ context.Context, label string, ns rstypes.NamespaceWithPermission) error {
	kub.log.WithFields(logrus.Fields{
		"name":   ns.ID,
		"cpu":    ns.CPU,
		"memory": ns.RAM,
		"label":  label,
		"access": ns.AccessLevel,
	}).Infoln("create namespace")
	return nil
}

func (kub kubeDummy) DeleteNamespace(_ context.Context, ns rstypes.Namespace) error {
	kub.log.WithField("name", ns.ID).Infoln("delete namespace")
	return nil
}

func (kub kubeDummy) SetNamespaceQuota(_ context.Context, label string, ns rstypes.NamespaceWithPermission) error {
	kub.log.WithFields(logrus.Fields{
		"name":   ns.ID,
		"cpu":    ns.CPU,
		"memory": ns.RAM,
		"label":  label,
		"access": ns.AccessLevel,
	}).Infoln("set namespace quota")
	return nil
}

func (kubeDummy) String() string {
	return "kube api dummy"
}
