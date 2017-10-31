package other

import (
	"context"
	"net/http"
	"net/url"
	"fmt"
)

type Kube interface {
	CreateNamespace(ctx context.Context, name string, cpu, memory uint) error
	DeleteNamespace(ctx context.Context, name string) error
}

type kube struct {
	c *http.Client
	u *url.URL
}

func NewKube(u *url.URL) Kube {
	k := &kube{
		c: http.DefaultClient,
		u: u,
	}
	return k
}

func (kub kube) CreateNamespace(ctx context.Context, name string, cpu, memory uint) error {
	refURL := &url.URL{
		Path:     "/api/v1/namespaces",
		RawQuery: fmt.Sprintf("cpu=%d&memory=%d", cpu, memory),
	}
	req, err := http.NewRequest("POST", kub.u.ResolveReference(refURL), nil) // FIXME: FATAL ERROR
	if err != nil {
		//
	}

	return nil
}

func (kub kube) DeleteNamespace(ctx context.Context, name string) error {
	return nil
}
