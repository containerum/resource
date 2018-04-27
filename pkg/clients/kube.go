package clients

import (
	"context"
	"fmt"
	"net/url"

	kubtypesInternal "git.containerum.net/ch/kube-api/pkg/model"
	"github.com/containerum/cherry"
	"github.com/containerum/cherry/adaptors/cherrylog"
	"github.com/containerum/kube-client/pkg/cherry/resource-service"
	kubtypes "github.com/containerum/kube-client/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
)

// Kube is an interface to kube-api service
type Kube interface {
	CreateNamespace(ctx context.Context, ns kubtypesInternal.NamespaceWithOwner) error
	SetNamespaceQuota(ctx context.Context, ns kubtypesInternal.NamespaceWithOwner) error
	DeleteNamespace(ctx context.Context, label string) error

	CreateDeployment(ctx context.Context, nsID string, deploy kubtypesInternal.DeploymentWithOwner) error
	DeleteDeployment(ctx context.Context, nsID, deplName string) error
	ReplaceDeployment(ctx context.Context, nsID string, deploy kubtypesInternal.DeploymentWithOwner) error
	SetDeploymentReplicas(ctx context.Context, nsID, deplName string, replicas int) error
	SetContainerImage(ctx context.Context, nsID, deplName string, container kubtypes.UpdateImage) error

	CreateIngress(ctx context.Context, nsID string, ingress kubtypesInternal.IngressWithOwner) error
	DeleteIngress(ctx context.Context, nsID, ingressName string) error

	CreateSecret(ctx context.Context, nsID string, secret kubtypesInternal.SecretWithOwner) error
	DeleteSecret(ctx context.Context, nsID, secretName string) error

	CreateService(ctx context.Context, nsID string, service kubtypesInternal.ServiceWithOwner) error
	UpdateService(ctx context.Context, nsID string, service kubtypesInternal.ServiceWithOwner) error
	DeleteService(ctx context.Context, nsID, serviceName string) error
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
		SetError(cherry.Err{}).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json")
	client.JSONMarshal = jsoniter.Marshal
	client.JSONUnmarshal = jsoniter.Unmarshal
	return kube{
		client: client,
		log:    cherrylog.NewLogrusAdapter(log),
	}
}

func (kub kube) CreateNamespace(ctx context.Context, ns kubtypesInternal.NamespaceWithOwner) error {
	kub.log.WithFields(logrus.Fields{
		"cpu":    ns.Resources.Hard.CPU,
		"memory": ns.Resources.Hard.Memory,
		"name":   ns.Label,
		"access": ns.Access,
	}).Debug("create namespace")

	resp, err := kub.client.R().
		SetBody(ns).
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		Post("/namespaces")
	if err != nil {
		return rserrors.ErrInternal().Log(err, kub.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}
	return nil
}

func (kub kube) DeleteNamespace(ctx context.Context, label string) error {
	kub.log.WithField("label", label).Debug("delete namespace")

	resp, err := kub.client.R().
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		Delete("/namespaces/" + url.PathEscape(label))
	if err != nil {
		return rserrors.ErrInternal().Log(err, kub.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}
	return nil
}

func (kub kube) SetNamespaceQuota(ctx context.Context, ns kubtypesInternal.NamespaceWithOwner) error {
	kub.log.WithFields(logrus.Fields{
		"cpu":    ns.Resources.Hard.CPU,
		"memory": ns.Resources.Hard.Memory,
		"label":  ns.Label,
	}).Debug("set namespace quota")

	resp, err := kub.client.R().
		SetBody(ns).
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		Put("/namespaces/" + url.PathEscape(ns.Label))
	if err != nil {
		return rserrors.ErrInternal().Log(err, kub.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}
	return nil
}

func (kub kube) CreateDeployment(ctx context.Context, nsID string, deploy kubtypesInternal.DeploymentWithOwner) error {
	kub.log.WithField("ns_id", nsID).Debug("create deployment %+v", deploy)

	resp, err := kub.client.R().
		SetBody(deploy).
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		Post(fmt.Sprintf("/namespaces/%s/deployments", nsID))
	if err != nil {
		return rserrors.ErrInternal().Log(err, kub.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}
	return nil
}

func (kub kube) DeleteDeployment(ctx context.Context, nsID, deplName string) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id":       nsID,
		"deploy_name": deplName,
	}).Debug("delete deployment")

	resp, err := kub.client.R().
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		Delete(fmt.Sprintf("/namespaces/%s/deployments/%s", nsID, deplName))
	if err != nil {
		return rserrors.ErrInternal().Log(err, kub.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}
	return nil
}

func (kub kube) ReplaceDeployment(ctx context.Context, nsID string, deploy kubtypesInternal.DeploymentWithOwner) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id": nsID,
	}).Debug("replace deployment %+v", deploy)

	resp, err := kub.client.R().
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		SetBody(deploy).
		Put(fmt.Sprintf("/namespaces/%s/deployments/%s", nsID, deploy.Name))
	if err != nil {
		return rserrors.ErrInternal().Log(err, kub.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}
	return nil
}

func (kub kube) SetDeploymentReplicas(ctx context.Context, nsID, deplName string, replicas int) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id":       nsID,
		"deploy_name": deplName,
		"replicas":    replicas,
	}).Debug("change replicas")

	resp, err := kub.client.R().
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		SetBody(kubtypes.UpdateReplicas{Replicas: replicas}).
		Put(fmt.Sprintf("/namespaces/%s/deployments/%s/replicas", nsID, deplName))
	if err != nil {
		return rserrors.ErrInternal().Log(err, kub.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}
	return nil
}

func (kub kube) SetContainerImage(ctx context.Context, nsID, deplName string, container kubtypes.UpdateImage) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id":       nsID,
		"deploy_name": deplName,
		"container":   container.Container,
		"image":       container.Image,
	}).Debug("set container image")

	resp, err := kub.client.R().
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		SetBody(container).
		Put(fmt.Sprintf("/namespaces/%s/deployments/%s/image", nsID, deplName))
	if err != nil {
		return rserrors.ErrInternal().Log(err, kub.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}
	return nil
}

func (kub kube) CreateIngress(ctx context.Context, nsID string, ingress kubtypesInternal.IngressWithOwner) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id": nsID,
	}).Debugf("create ingress %+v", ingress)

	resp, err := kub.client.R().
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		SetBody(ingress).
		Post(fmt.Sprintf("/namespaces/%s/ingresses", nsID))
	if err != nil {
		return rserrors.ErrInternal().Log(err, kub.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}
	return nil
}

func (kub kube) DeleteIngress(ctx context.Context, nsID, ingressName string) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id":        nsID,
		"ingress_name": ingressName,
	}).Debug("delete ingress")

	resp, err := kub.client.R().
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		Delete(fmt.Sprintf("/namespaces/%s/ingresses/%s", nsID, ingressName))
	if err != nil {
		return rserrors.ErrInternal().Log(err, kub.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}
	return nil
}

func (kub kube) CreateSecret(ctx context.Context, nsID string, secret kubtypesInternal.SecretWithOwner) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id": nsID,
	}).Debugf("create secret %+v", secret)

	resp, err := kub.client.R().
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		SetBody(secret).
		Post(fmt.Sprintf("/namespaces/%s/secrets", nsID))
	if err != nil {
		return rserrors.ErrInternal().Log(err, kub.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}
	return nil
}

func (kub kube) DeleteSecret(ctx context.Context, nsID, secretName string) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id":       nsID,
		"secret_name": secretName,
	}).Debug("delete secret")

	resp, err := kub.client.R().
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		Delete(fmt.Sprintf("/namespaces/%s/secrets/%s", nsID, secretName))
	if err != nil {
		return rserrors.ErrInternal().Log(err, kub.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}

	return nil
}

func (kub kube) CreateService(ctx context.Context, nsID string, service kubtypesInternal.ServiceWithOwner) error {
	kub.log.WithField("ns_id", nsID).Debugf("create service %+v", service)

	resp, err := kub.client.R().
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		SetBody(service).
		Post(fmt.Sprintf("/namespaces/%s/services", nsID))

	if err != nil {
		return rserrors.ErrInternal().Log(err, kub.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}

	return nil
}

func (kub kube) UpdateService(ctx context.Context, nsID string, service kubtypesInternal.ServiceWithOwner) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id":        nsID,
		"service_name": service.Name,
	}).Debugf("update service to %+v", service)

	resp, err := kub.client.R().
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		SetBody(service).
		Put(fmt.Sprintf("/namespaces/%s/services/%s", nsID, service.Name))

	if err != nil {
		return rserrors.ErrInternal().Log(err, kub.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}

	return nil
}

func (kub kube) DeleteService(ctx context.Context, nsID, serviceName string) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id":        nsID,
		"service_name": serviceName,
	}).Debug("delete service")

	resp, err := kub.client.R().
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		Delete(fmt.Sprintf("/namespaces/%s/services/%s", nsID, serviceName))

	if err != nil {
		return rserrors.ErrInternal().Log(err, kub.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}

	return nil
}

func (kub kube) String() string {
	return fmt.Sprintf("kube api http client: url=%v", kub.client.HostURL)
}

// Dummy implementation

type kubeDummy struct {
	log *logrus.Entry
}

// NewDummyKube creates a dummy client to kube-api service. It does nothing but logs actions.
func NewDummyKube() Kube {
	return kubeDummy{log: logrus.WithField("component", "kube_stub")}
}

func (kub kubeDummy) CreateNamespace(_ context.Context, ns kubtypesInternal.NamespaceWithOwner) error {
	kub.log.WithFields(logrus.Fields{
		"cpu":    ns.Resources.Hard.CPU,
		"memory": ns.Resources.Hard.Memory,
		"name":   ns.Label,
		"access": ns.Access,
	}).Debug("create namespace")
	return nil
}

func (kub kubeDummy) DeleteNamespace(_ context.Context, label string) error {
	kub.log.WithField("label", label).Debug("delete namespace")
	return nil
}

func (kub kubeDummy) SetNamespaceQuota(_ context.Context, ns kubtypesInternal.NamespaceWithOwner) error {
	kub.log.WithFields(logrus.Fields{
		"cpu":    ns.Resources.Hard.CPU,
		"memory": ns.Resources.Hard.Memory,
		"label":  ns.Label,
	}).Debug("set namespace quota")

	return nil
}

func (kub kubeDummy) CreateDeployment(_ context.Context, nsID string, deploy kubtypesInternal.DeploymentWithOwner) error {
	kub.log.WithField("ns_id", nsID).Debug("create deployment %+v", deploy)

	return nil
}

func (kub kubeDummy) DeleteDeployment(_ context.Context, nsID, deplName string) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id":       nsID,
		"deploy_name": deplName,
	}).Debug("delete deployment")

	return nil
}

func (kub kubeDummy) ReplaceDeployment(_ context.Context, nsID string, deploy kubtypesInternal.DeploymentWithOwner) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id":       nsID,
		"deploy_name": deploy.Name,
	}).Debug("replace deployment %+v", deploy)

	return nil
}

func (kub kubeDummy) SetDeploymentReplicas(ctx context.Context, nsID, deplName string, replicas int) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id":       nsID,
		"deploy_name": deplName,
		"replicas":    replicas,
	}).Debug("change replicas")

	return nil
}

func (kub kubeDummy) SetContainerImage(ctx context.Context, nsID, deplName string, container kubtypes.UpdateImage) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id":       nsID,
		"deploy_name": deplName,
		"container":   container.Container,
		"image":       container.Image,
	}).Debug("set container image")

	return nil
}

func (kub kubeDummy) CreateIngress(ctx context.Context, nsID string, ingress kubtypesInternal.IngressWithOwner) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id": nsID,
	}).Debugf("create ingress %+v", ingress)

	return nil
}

func (kub kubeDummy) DeleteIngress(ctx context.Context, nsID, ingressName string) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id":        nsID,
		"ingress_name": ingressName,
	}).Debug("delete ingress")

	return nil
}

func (kub kubeDummy) CreateSecret(ctx context.Context, nsID string, secret kubtypesInternal.SecretWithOwner) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id": nsID,
	}).Debugf("create secret %+v", secret)

	return nil
}

func (kub kubeDummy) DeleteSecret(ctx context.Context, nsID, secretName string) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id":       nsID,
		"secret_name": secretName,
	}).Debug("delete secret")

	return nil
}

func (kub kubeDummy) CreateService(ctx context.Context, nsID string, service kubtypesInternal.ServiceWithOwner) error {
	kub.log.WithField("ns_id", nsID).Debugf("create service %+v", service)

	return nil
}

func (kub kubeDummy) UpdateService(ctx context.Context, nsID string, service kubtypesInternal.ServiceWithOwner) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id":        nsID,
		"service_name": service.Name,
	}).Debugf("update service to %+v", service)

	return nil
}

func (kub kubeDummy) DeleteService(ctx context.Context, nsID, serviceName string) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id":        nsID,
		"service_name": serviceName,
	}).Debug("delete service")

	return nil
}

func (kubeDummy) String() string {
	return "kube api dummy"
}
