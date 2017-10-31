package other

import (
	"net/http"
	"net/url"
)

type VolumeSvc interface {
	CreateVolume() error
	DeleteVolume() error
}

type volumeSvcHTTP struct {
	c *http.Client
	u *url.URL
}

func NewVolumeSvc(u *url.URL) VolumeSvc {
	return volumeSvcHTTP{
		c: http.DefaultClient,
		u: u,
	}
}

func (vs volumeSvcHTTP) CreateVolume() error {
	return nil
}

func (vs volumeSvcHTTP) DeleteVolume() error {
	return nil
}
