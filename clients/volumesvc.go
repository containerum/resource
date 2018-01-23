package clients

import (
	"fmt"
	"net/url"

	"git.containerum.net/ch/json-types/errors"
	"github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
)

type VolumeSvc interface {
	CreateVolume() error
	DeleteVolume() error
}

type volumeSvcHTTP struct {
	client *resty.Client
	log    *logrus.Entry
}

func NewVolumeSvcHTTP(u *url.URL) VolumeSvc {
	log := logrus.WithField("component", "volume_client")
	client := resty.New().
		SetHostURL(u.String()).
		SetLogger(log.WriterLevel(logrus.DebugLevel)).
		SetDebug(true).
		SetError(errors.Error{})
	client.JSONMarshal = jsoniter.Marshal
	client.JSONUnmarshal = jsoniter.Unmarshal
	return volumeSvcHTTP{
		client: client,
		log:    log,
	}
}

func (vs volumeSvcHTTP) CreateVolume() error {
	return fmt.Errorf("not implemented")
}

func (vs volumeSvcHTTP) DeleteVolume() error {
	return fmt.Errorf("not implemented")
}

func (vs volumeSvcHTTP) String() string {
	return fmt.Sprintf("volume service http client: url=%v", vs.client.HostURL)
}

type volumeSvcStub struct {
	log *logrus.Entry
}

func NewVolumeSvcStub() VolumeSvc {
	return volumeSvcStub{log: logrus.WithField("component", "volume_stub")}
}

func (v volumeSvcStub) CreateVolume() error {
	v.log.Infoln("volume created")
	return nil
}

func (v volumeSvcStub) DeleteVolume() error {
	v.log.Infoln("volume deleted")
	return nil
}

func (volumeSvcStub) String() string {
	return "volume svc dummy"
}
