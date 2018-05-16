package clients

import (
	"context"

	"fmt"

	"github.com/containerum/cherry"
	kubtypes "github.com/containerum/kube-client/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/go-resty/resty"
	"github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
)

type Permissions interface {
	GetNamespaceLimits(ctx context.Context, namespaceID string) (kubtypes.Namespace, error)
}

type permissions struct {
	resty  *resty.Client
	logger logrus.FieldLogger
}

func NewPermissionsHTTP(permissionsHost string) Permissions {
	var client = resty.New().
		SetHostURL(permissionsHost).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json")
	client.JSONMarshal = jsoniter.Marshal
	client.JSONUnmarshal = jsoniter.Unmarshal
	return permissions{
		logger: logrus.WithField("component", "permissions_client"),
		resty:  client,
	}
}

func (client permissions) GetNamespaceLimits(ctx context.Context, namespaceID string) (kubtypes.Namespace, error) {
	client.logger.
		WithField("namespace_id", namespaceID).
		Debugf("getting namespace limits")
	var ns kubtypes.Namespace
	var errResult cherry.Err
	_, err := client.resty.R().
		SetContext(ctx).
		SetResult(&ns).
		SetError(&errResult).
		SetPathParams(map[string]string{
			"namespace": namespaceID,
		}).SetHeaders(httputil.RequestXHeadersMap(ctx)).
		Get("/namespaces/{namespace}")

	fmt.Println("TEST", ns.Resources.Hard)
	fmt.Println("TEST", ns.Resources.Used)
	return ns, func() error {
		if err != nil {
			return err
		}
		if errResult.ID != (cherry.ErrID{}) {
			return &errResult
		}
		return nil
	}()
}
