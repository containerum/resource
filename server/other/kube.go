package other

import (
	"net/http"
	"net/url"
)

type Kube interface {
	CreateNamespace(name string, cpu, memory uint) error
	DeleteNamespace(name string) error
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

func (kube kube) CreateNamespace(name string, cpu, memory uint) error {
	return nil
}

func (kube kube) DeleteNamespace(name string) error {
	return nil
}
