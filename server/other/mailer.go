package other

import (
	"net/http"
	"net/url"

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
	logrus.Warnf("Mailer.SendNamespaceCreated userID=%s nsLabel=%s", userID, nsLabel)
	return nil
}

func (ml mailerHTTP) SendNamespaceDeleted(userID, nsLabel string) error {
	logrus.Warnf("Mailer.SendNamespaceDeleted userID=%s nsLabel=%s", userID, nsLabel)
	return nil
}
