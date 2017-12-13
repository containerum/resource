package other

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"bitbucket.org/exonch/resource-service/server/model"

	"github.com/sirupsen/logrus"
)

type Mailer interface {
	SendNamespaceCreated(userID, nsLabel string, t model.NamespaceTariff) error
	SendNamespaceDeleted(userID, nsLabel string, t model.NamespaceTariff) error

	SendVolumeCreated(userID, nsLabel string, t model.VolumeTariff) error
	SendVolumeDeleted(userID, nsLabel string, t model.VolumeTariff) error
}

type mailerHTTP struct {
	c *http.Client
	u *url.URL
}

type mailerSendRequest struct {
	Template  string                 `json:"template"`
	UserID    string                 `json:"user_id"`
	Variables map[string]interface{} `json:"variables"`
}

type mailerSendResponse struct {
	Error  string `json:"error,omitempty"`

	UserID string `json:"user_id"`
}

func NewMailerHTTP(u *url.URL) Mailer {
	return mailerHTTP{
		c: http.DefaultClient,
		u: u,
	}
}

func (ml mailerHTTP) sendRequest(eventName string, userID string, vars map[string]interface{}) error {
	var reqObj = mailerSendRequest{
		Template:  eventName,
		UserID:    userID,
		Variables: vars,
	}
	var respObj mailerSendResponse

	reqBytes, _ := json.Marshal(reqObj)
	reqBuf := bytes.NewBuffer(reqBytes)
	reqURL, _ := ml.u.Parse("/templates/namespace_created/send")
	httpReq, _ := http.NewRequest(http.MethodPost, reqURL.String(), reqBuf)
	httpResp, err := ml.c.Do(httpReq)
	if err != nil {
		return err
	}
	respBytes, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(respBytes, &respObj)
	if err != nil {
		return err
	}
	if respObj.Error != "" || respObj.UserID == "" {
		return fmt.Errorf("http status %s: error %s", httpResp.Status, respObj.Error)
	}

	return nil
}

func (ml mailerHTTP) SendNamespaceCreated(userID, nsLabel string, t model.NamespaceTariff) error {
	err := ml.sendRequest("ns_created", userID, map[string]interface{}{
		"NAMESPACE": nsLabel,
		"CPU": *t.CpuLimit,
		"RAM": *t.MemoryLimit,
		"DAILY_PAY": *t.Price,
		//"DAILY_PAY_TOTAL": 0, // FIXME
		//"STORAGE": 0, // FIXME
	})
	if err != nil {
		return err
	}
	return nil
}

func (ml mailerHTTP) SendNamespaceDeleted(userID, nsLabel string, t model.NamespaceTariff) error {
	err := ml.sendRequest("ns_deleted", userID, map[string]interface{}{
		"NAMESPACE": nsLabel,
	})
	if err != nil {
		return err
	}
	return nil
}

func (ml mailerHTTP) SendVolumeCreated(userID, volLabel string, t model.VolumeTariff) error {
	err := ml.sendRequest("vol_created", userID, map[string]interface{}{
		"VOLUME": volLabel,
		"STORAGE": *t.StorageLimit,
		"DAILY_PAY": *t.Price,
		//"DAILY_PAY_TOTAL": 0, // FIXME
	})
	if err != nil {
		return err
	}
	return nil
}

func (ml mailerHTTP) SendVolumeDeleted(userID, volLabel string, t model.VolumeTariff) error {
	err := ml.sendRequest("vol_deleted", userID, map[string]interface{}{
		"VOLUME": volLabel,
	})
	if err != nil {
		return err
	}
	return nil
}

func (m mailerHTTP) String() string {
	return fmt.Sprintf("mail service http client: url=%v", m.u)
}

type mailerStub struct{}

func NewMailerStub() Mailer {
	return mailerStub{}
}

func (mailerStub) SendNamespaceCreated(userID, nsLabel string, t model.NamespaceTariff) error {
	logrus.Infof("Mailer.SendNamespaceCreated userID=%s nsLabel=%s tariff=%+v", userID, nsLabel, t)
	return nil
}

func (mailerStub) SendNamespaceDeleted(userID, nsLabel string, t model.NamespaceTariff) error {
	logrus.Infof("Mailer.SendNamespaceDeleted userID=%s nsLabel=%s tariff=%+v", userID, nsLabel, t)
	return nil
}

func (mailerStub) SendVolumeCreated(userID, label string, t model.VolumeTariff) error {
	logrus.Infof("Mailer.SendVolumeCreated userID=%s label=%s tariff=%+v", userID, label, t)
	return nil
}

func (mailerStub) SendVolumeDeleted(userID, label string, t model.VolumeTariff) error {
	logrus.Infof("Mailer.SendVolumeDeleted userID=%s label=%s tariff=%+v", userID, label, t)
	return nil
}

func (mailerStub) String() string {
	return "mail service dummy"
}
