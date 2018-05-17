package clients

import (
	"context"

	"github.com/json-iterator/go"
	"github.com/sirupsen/logrus"

	permtypes "git.containerum.net/ch/permissions/pkg/model"
	"github.com/containerum/cherry"
	kubtypes "github.com/containerum/kube-client/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/go-resty/resty"
)

type Permissions interface {
	GetNamespaceLimits(ctx context.Context, namespaceID string) (kubtypes.Namespace, error)
}

type permissions struct {
	resty  *resty.Client
	logger logrus.FieldLogger
}

func NewPermissionsHTTP(permissionsHost string) Permissions {
	log := logrus.WithField("component", "permissions_client")
	var client = resty.New().
		SetHostURL(permissionsHost).
		SetLogger(log.WriterLevel(logrus.DebugLevel)).
		SetDebug(true).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json")
	client.JSONMarshal = jsoniter.Marshal
	client.JSONUnmarshal = jsoniter.Unmarshal
	return permissions{
		logger: log,
		resty:  client,
	}
}

func (client permissions) GetNamespaceLimits(ctx context.Context, namespaceID string) (kubtypes.Namespace, error) {
	client.logger.
		WithField("namespace_id", namespaceID).
		Debugf("getting namespace limits")
	var ns permtypes.Namespace
	var errResult cherry.Err
	_, err := client.resty.R().
		SetContext(ctx).
		SetResult(&ns).
		SetError(&errResult).
		SetPathParams(map[string]string{
			"namespace": namespaceID,
		}).SetHeaders(httputil.RequestXHeadersMap(ctx)).
		Get("/namespaces/{namespace}")

	maxint := uint(ns.MaxIntServices)
	maxext := uint(ns.MaxExtServices)

	ret := kubtypes.Namespace{
		MaxIntService: &maxint,
		MaxExtService: &maxext,
		Resources: kubtypes.Resources{
			Hard: kubtypes.Resource{
				CPU:    uint(ns.CPU),
				Memory: uint(ns.RAM),
			},
		},
	}

	return ret, func() error {
		if err != nil {
			return err
		}
		if errResult.ID != (cherry.ErrID{}) {
			return &errResult
		}
		return nil
	}()
}
