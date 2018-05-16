package permissions

import (
	"github.com/containerum/cherry"
	"github.com/containerum/kube-client/pkg/model"
	"github.com/go-resty/resty"
)

type Client struct {
	resty *resty.Client
}

func NewClient(permissionsHost string) *Client {
	return &Client{
		resty: resty.New().
			SetHostURL(permissionsHost),
	}
}

func (client *Client) GetLimits(userRole, userID, namespaceID string) (model.Namespace, error) {
	var ns model.Namespace
	var errResult cherry.Err
	_, err := client.resty.R().
		SetResult(&ns).
		SetError(&errResult).
		SetPathParams(map[string]string{
			"namespace": namespaceID,
		}).SetHeader("X-User-Role", userRole).
		SetHeader("X-User-ID", userID).
		Get("/namespaces/{namespace}")
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
