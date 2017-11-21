package other

import (
	"net/http"
	"net/url"
	"fmt"

	"github.com/sirupsen/logrus"
)

type Mailer interface {
	SendNamespaceCreated(userID, nsLabel string) error
	SendNamespaceDeleted(userID, nsLabel string) error
}

type mailerHTTP struct {
	c *http.Client
	u *url.URL
}

func NewMailerHTTP(u *url.URL) Mailer {
	return mailerHTTP{
		c: http.DefaultClient,
		u: u,
	}
}

func (ml mailerHTTP) SendNamespaceCreated(userID, nsLabel string) error {
	return fmt.Errorf("not implemented")
}

func (ml mailerHTTP) SendNamespaceDeleted(userID, nsLabel string) error {
	return fmt.Errorf("not implemented")
}

type mailerStub struct{}

func NewMailerStub() Mailer {
	return mailerStub{}
}

func (mailerStub) SendNamespaceCreated(userID, nsLabel string) error {
	logrus.Infof("Mailer.SendNamespaceCreated userID=%s nsLabel=%s", userID, nsLabel)
	return nil
}

func (mailerStub) SendNamespaceDeleted(userID, nsLabel string) error {
	logrus.Infof("Mailer.SendNamespaceDeleted userID=%s nsLabel=%s", userID, nsLabel)
	return nil
}
