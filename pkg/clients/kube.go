package clients

import (
	"context"
	"fmt"
	"net/url"

	"git.containerum.net/ch/resource-service/pkg/rsErrors"
	"github.com/containerum/cherry"
	"github.com/containerum/cherry/adaptors/cherrylog"
	kubtypes "github.com/containerum/kube-client/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
)

// Kube is an interface to kube-api service
type Kube interface {
	CreateDeployment(ctx context.Context, nsID string, deploy kubtypes.Deployment) error
	DeleteDeployment(ctx context.Context, nsID, deplName string) error
	UpdateDeployment(ctx context.Context, nsID string, deploy kubtypes.Deployment) error
	SetDeploymentReplicas(ctx context.Context, nsID, deplName string, replicas int) error
	SetContainerImage(ctx context.Context, nsID, deplName string, container kubtypes.UpdateImage) error

	CreateIngress(ctx context.Context, nsID string, ingress kubtypes.Ingress) error
	UpdateIngress(ctx context.Context, nsID string, ingress kubtypes.Ingress) error
	DeleteIngress(ctx context.Context, nsID, ingressName string) error

	CreateSecret(ctx context.Context, nsID string, secret kubtypes.Secret) error
	DeleteSecret(ctx context.Context, nsID, secretName string) error

	CreateService(ctx context.Context, nsID string, service kubtypes.Service) error
	UpdateService(ctx context.Context, nsID string, service kubtypes.Service) error
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

func (kub kube) CreateDeployment(ctx context.Context, nsID string, deploy kubtypes.Deployment) error {
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

func (kub kube) UpdateDeployment(ctx context.Context, nsID string, deploy kubtypes.Deployment) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id": nsID,
	}).Debug("update deployment %+v", deploy)

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

func (kub kube) CreateIngress(ctx context.Context, nsID string, ingress kubtypes.Ingress) error {
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

func (kub kube) UpdateIngress(ctx context.Context, nsID string, ingress kubtypes.Ingress) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id":        nsID,
		"ingress_name": ingress.Name,
	}).Debugf("update ingress to %+v", ingress)

	resp, err := kub.client.R().
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		SetBody(ingress).
		Put(fmt.Sprintf("/namespaces/%s/ingresses/%s", nsID, ingress.Name))
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

func (kub kube) CreateSecret(ctx context.Context, nsID string, secret kubtypes.Secret) error {
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

func (kub kube) CreateService(ctx context.Context, nsID string, service kubtypes.Service) error {
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

func (kub kube) UpdateService(ctx context.Context, nsID string, service kubtypes.Service) error {
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

func (kub kubeDummy) CreateDeployment(_ context.Context, nsID string, deploy kubtypes.Deployment) error {
	kub.log.WithField("ns_id", nsID).Debug("create deployment %+v", deploy)

	return nil
}

func (kub kubeDummy) UpdateDeployment(ctx context.Context, nsID string, deploy kubtypes.Deployment) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id": nsID,
	}).Debug("update deployment %+v", deploy)

	return nil
}

func (kub kubeDummy) DeleteDeployment(ctx context.Context, nsID, deplName string) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id":       nsID,
		"deploy_name": deplName,
	}).Debug("delete deployment")

	return nil
}

func (kub kubeDummy) ReplaceDeployment(ctx context.Context, nsID string, deploy kubtypes.Deployment) error {
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

func (kub kubeDummy) CreateIngress(ctx context.Context, nsID string, ingress kubtypes.Ingress) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id": nsID,
	}).Debugf("create ingress %+v", ingress)

	return nil
}

func (kub kubeDummy) UpdateIngress(ctx context.Context, nsID string, ingress kubtypes.Ingress) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id":        nsID,
		"ingress_name": ingress.Name,
	}).Debugf("update ingress to %+v", ingress)

	return nil
}

func (kub kubeDummy) DeleteIngress(ctx context.Context, nsID, ingressName string) error {
	kub.log.WithFields(logrus.Fields{
		"ns_id":        nsID,
		"ingress_name": ingressName,
	}).Debug("delete ingress")

	return nil
}

func (kub kubeDummy) CreateSecret(ctx context.Context, nsID string, secret kubtypes.Secret) error {
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

func (kub kubeDummy) CreateService(ctx context.Context, nsID string, service kubtypes.Service) error {
	kub.log.WithField("ns_id", nsID).Debugf("create service %+v", service)

	return nil
}

func (kub kubeDummy) UpdateService(ctx context.Context, nsID string, service kubtypes.Service) error {
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
