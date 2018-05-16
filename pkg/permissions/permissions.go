package permissions

import (
	"git.containerum.net/ch/kube-client/pkg/model"
	"github.com/go-resty/resty"
)

type Client struct {
	role   string
	userID string
	resty  *resty.Request
}

func NewClient(permissionsHost, userRole, userID string) *Client {
	return &Client{
		role:   userRole,
		userID: userID,
		resty: resty.New().
			SetHostURL(permissionsHost).
			SetHeader("X-User-Role", userRole).
			SetHeader("X-User-ID", userID).
			R(),
	}
}

func (client *Client) GetLimits(namespaceID string) (model.Resources, error) {
	var ns model.Namespace
	_, err := client.resty.
		SetResult(&ns).
		SetPathParams(map[string]string{
		"namespace": namespaceID,
	}).
		Get("/namespaces/{namespace}")
	return ns.Resources, err
}
