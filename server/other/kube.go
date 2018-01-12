package other

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"

	"git.containerum.net/ch/json-types/errors"
	"github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
)

type Kube interface {
	CreateNamespace(ctx context.Context, name string, cpu, memory uint, label, access string) error
	SetNamespaceQuota(ctx context.Context, name string, cpu, memory uint, label, access string) error
	DeleteNamespace(ctx context.Context, name string) error
}

type kube struct {
	client *resty.Client
	log    *logrus.Entry
}

const namespaceHeader = "x-user-namespace"

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
	ID     string `json:"id,omitempty"`
	Label  string `json:"label,omitempty"`
	Access string `json:"access,omitempty"`
}

func (kub kube) createNamespaceHeaderValue(e kubNamespaceHeaderElement) string {
	xUserNamespaceBytes, _ := kub.client.JSONMarshal([]kubNamespaceHeaderElement{e})
	return base64.StdEncoding.EncodeToString(xUserNamespaceBytes)
}

func (kub kube) CreateNamespace(ctx context.Context, name string, cpu, memory uint, label, access string) error {
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

func (kub kube) SetNamespaceQuota(ctx context.Context, name string, cpu, memory uint, label, access string) (err error) {
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

type kubeStub struct {
	log *logrus.Entry
}

func NewKubeStub() Kube {
	return kubeStub{log: logrus.WithField("component", "kube_stub")}
}

func (kub kubeStub) CreateNamespace(_ context.Context, name string, cpu, memory uint, label, access string) error {
	kub.log.WithFields(logrus.Fields{
		"name":   name,
		"cpu":    cpu,
		"memory": memory,
		"label":  label,
		"access": access,
	}).Infoln("create namespace")
	return nil
}

func (kub kubeStub) DeleteNamespace(_ context.Context, name string) error {
	kub.log.WithField("name", name).Infoln("delete namespace")
	return nil
}

func (kub kubeStub) SetNamespaceQuota(_ context.Context, name string, cpu, memory uint, label, access string) error {
	kub.log.WithFields(logrus.Fields{
		"name":   name,
		"cpu":    cpu,
		"memory": memory,
		"label":  label,
		"access": access,
	}).Infoln("set namespace quota")
	return nil
}

func (kubeStub) String() string {
	return "kube api dummy"
}
