package other

import (
	"net/http"
	"net/url"
	"fmt"

	"github.com/sirupsen/logrus"
)

type VolumeSvc interface {
	CreateVolume() error
	DeleteVolume() error
}

type volumeSvcHTTP struct {
	c *http.Client
	u *url.URL
}

func NewVolumeSvcHTTP(u *url.URL) VolumeSvc {
	return volumeSvcHTTP{
		c: http.DefaultClient,
		u: u,
	}
}

func (vs volumeSvcHTTP) CreateVolume() error {
	return fmt.Errorf("not implemented")
}

func (vs volumeSvcHTTP) DeleteVolume() error {
	return fmt.Errorf("not implemented")
}

func (v volumeSvcHTTP) String() string {
	return fmt.Sprintf("volume service http client: url=%v", v.u)
}

type volumeSvcStub struct{}

func NewVolumeSvcStub() VolumeSvc {
	return volumeSvcStub{}
}

func (volumeSvcStub) CreateVolume() error {
	logrus.Infof("volumeSvcStub.CreateVolume")
	return nil
}

func (volumeSvcStub) DeleteVolume() error {
	logrus.Infof("volumeSvcStub.DeleteVolume")
	return nil
}

func (volumeSvcStub) String() string {
	return "volume svc dummy"
}
