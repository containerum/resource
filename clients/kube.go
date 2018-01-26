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
	CreateNamespace(ctx context.Context, name string, cpu, memory int, label string, access rstypes.PermissionStatus) error
	SetNamespaceQuota(ctx context.Context, name string, cpu, memory int, label string, access rstypes.PermissionStatus) error
	DeleteNamespace(ctx context.Context, name string) error
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

func (kub kube) CreateNamespace(ctx context.Context, name string, cpu, memory int, label string, access rstypes.PermissionStatus) error {
	kub.log.WithFields(logrus.Fields{
		"name":   name,
		"cpu":    cpu,
		"memory": memory,
		"label":  label,
		"access": access,
	}).Infoln("create namespace")

	resp, err := kub.client.R().SetBody(map[string]interface{}{
		"kind":       "Namespace",
		"apiVersion": "v1",
		"metadata": map[string]interface{}{
			"name": name,
		},
	}).
		SetQueryParam("cpu", fmt.Sprint(cpu)).
		SetQueryParam("memory", fmt.Sprint(memory)).
		SetHeader(namespaceHeader, kub.createNamespaceHeaderValue(kubNamespaceHeaderElement{
			ID:     name,
			Label:  label,
			Access: access,
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

func (kub kube) DeleteNamespace(ctx context.Context, name string) error {
	kub.log.WithField("name", name).Infoln("delete namespace")

	resp, err := kub.client.R().
		SetHeader(namespaceHeader, kub.createNamespaceHeaderValue(kubNamespaceHeaderElement{
			ID:     name,
			Label:  "unknown", // not important though
			Access: "owner",
		})).
		SetContext(ctx).
		Delete("api/v1/namespaces/" + url.PathEscape(name))
	if err != nil {
		return err
	}
	if resp.Error() != nil {
		return resp.Error().(*errors.Error)
	}
	return nil
}

func (kub kube) SetNamespaceQuota(ctx context.Context, name string, cpu, memory int, label string, access rstypes.PermissionStatus) (err error) {
	// TODO: update cpu and memory also

	kub.log.WithFields(logrus.Fields{
		"name":   name,
		"cpu":    cpu,
		"memory": memory,
		"label":  label,
		"access": access,
	}).Infoln("set namespace quota")

	resp, err := kub.client.R().
		SetHeader(namespaceHeader, kub.createNamespaceHeaderValue(kubNamespaceHeaderElement{
			ID:     name,
			Access: "owner",
		})).
		SetContext(ctx).
		Put("api/v1/namespaces/" + url.PathEscape(name) + "/resourcequotas/quota")
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

func (kub kubeDummy) CreateNamespace(_ context.Context, name string, cpu, memory int, label string, access rstypes.PermissionStatus) error {
	kub.log.WithFields(logrus.Fields{
		"name":   name,
		"cpu":    cpu,
		"memory": memory,
		"label":  label,
		"access": access,
	}).Infoln("create namespace")
	return nil
}

func (kub kubeDummy) DeleteNamespace(_ context.Context, name string) error {
	kub.log.WithField("name", name).Infoln("delete namespace")
	return nil
}

func (kub kubeDummy) SetNamespaceQuota(_ context.Context, name string, cpu, memory int, label string, access rstypes.PermissionStatus) error {
	kub.log.WithFields(logrus.Fields{
		"name":   name,
		"cpu":    cpu,
		"memory": memory,
		"label":  label,
		"access": access,
	}).Infoln("set namespace quota")
	return nil
}

func (kubeDummy) String() string {
	return "kube api dummy"
}
