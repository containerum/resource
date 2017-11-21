package other

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/sirupsen/logrus"
)

type AuthSvc interface {
	UpdateUserAccess(userID string) error // TODO
}

type authSvcHTTP struct {
	c *http.Client
	u *url.URL
}

func NewAuthSvcHTTP(u *url.URL) AuthSvc {
	return &authSvcHTTP{
		c: http.DefaultClient,
		u: u,
	}
}

func (as authSvcHTTP) UpdateUserAccess(userID string) error {
	return fmt.Errorf("not implemented")
}

type authSvcStub struct{}

func NewAuthSvcStub(...interface{}) AuthSvc {
	return authSvcStub{}
}

func (authSvcStub) UpdateUserAccess(userID string) error {
	logrus.Infof("authSvcStub.UpdateUserAccess userID=%s", userID)
	return nil
}
