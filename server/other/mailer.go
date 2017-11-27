package other

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/sirupsen/logrus"
)

type Mailer interface {
	SendNamespaceCreated(userID, nsLabel string) error
	SendNamespaceDeleted(userID, nsLabel string) error

	SendVolumeCreated(userID, nsLabel string) error
	SendVolumeDeleted(userID, nsLabel string) error
}

type mailerHTTP struct {
	c *http.Client
	u *url.URL
}

type mailerSendRequest struct {
	Delay   int `json:"delay"`
	Message struct {
		Subject         string            `json:"subject"`
		SenderEmail     string            `json:"sender_email"`
		SenderName      string            `json:"sender_name"`
		CommonVariables map[string]string `json:"common_variables"`
		Recipients      []interface{}     `json:"recipient_data"`
	} `json:"message"`
}

type mailerRecipientStruct struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Email     string            `json:"email"`
	Variables map[string]string `json:"variables"`
}

type mailerSendResponse struct {
	Statuses []struct {
		Recipient_id string
		Status       string
	}
}

func NewMailerHTTP(u *url.URL) Mailer {
	return mailerHTTP{
		c: http.DefaultClient,
		u: u,
	}
}

func (ml mailerHTTP) sendRequest(reqObj mailerSendRequest, recipObj mailerRecipientStruct) (err error) {
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
	if len(respObj.Statuses) == 0 {
		return fmt.Errorf("mailer error: no statuses")
	}
	if respObj.Statuses[0].Status != "" {
		return fmt.Errorf("mailer error: send failed")
	}

	return nil
}

func (ml mailerHTTP) SendNamespaceCreated(userID, nsLabel string) error {
	var reqObj mailerSendRequest
	var recipientObj mailerRecipientStruct

	reqObj.Message.Subject = fmt.Sprintf("Containerum: Namespace %s created", nsLabel)
	reqObj.Message.SenderEmail = "info@containerum.com"
	reqObj.Message.SenderName = ""
	reqObj.Message.CommonVariables = make(map[string]string)
	recipientObj.ID = userID
	recipientObj.Variables = make(map[string]string)
	recipientObj.Variables["namespace_label"] = nsLabel
	reqObj.Message.Recipients = append(reqObj.Message.Recipients, recipientObj)

	err := ml.sendRequest(reqObj, recipientObj)
	if err != nil {
		return err
	}
	return nil
}

func (ml mailerHTTP) SendNamespaceDeleted(userID, nsLabel string) error {
	var reqObj mailerSendRequest
	var recipientObj mailerRecipientStruct

	reqObj.Message.Subject = fmt.Sprintf("Containerum: Namespace %s deleted", nsLabel)
	reqObj.Message.SenderEmail = "info@containerum.com"
	reqObj.Message.SenderName = ""
	reqObj.Message.CommonVariables = make(map[string]string)
	recipientObj.ID = userID
	recipientObj.Variables = make(map[string]string)
	recipientObj.Variables["namespace_label"] = nsLabel
	reqObj.Message.Recipients = append(reqObj.Message.Recipients, recipientObj)

	err := ml.sendRequest(reqObj, recipientObj)
	if err != nil {
		return err
	}
	return nil
}

func (ml mailerHTTP) SendVolumeCreated(userID, volLabel string) error {
	var reqObj mailerSendRequest
	var recipientObj mailerRecipientStruct

	reqObj.Message.Subject = fmt.Sprintf("Containerum: Volume %s created", volLabel)
	reqObj.Message.SenderEmail = "info@containerum.com"
	reqObj.Message.SenderName = ""
	reqObj.Message.CommonVariables = make(map[string]string)
	recipientObj.ID = userID
	recipientObj.Variables = make(map[string]string)
	recipientObj.Variables["volume_label"] = volLabel
	reqObj.Message.Recipients = append(reqObj.Message.Recipients, recipientObj)

	err := ml.sendRequest(reqObj, recipientObj)
	if err != nil {
		return err
	}
	return nil
}

func (ml mailerHTTP) SendVolumeDeleted(userID, volLabel string) error {
	var reqObj mailerSendRequest
	var recipientObj mailerRecipientStruct

	reqObj.Message.Subject = fmt.Sprintf("Containerum: Volume %s deleted", volLabel)
	reqObj.Message.SenderEmail = "info@containerum.com"
	reqObj.Message.SenderName = ""
	reqObj.Message.CommonVariables = make(map[string]string)
	recipientObj.ID = userID
	recipientObj.Variables = make(map[string]string)
	recipientObj.Variables["volume_label"] = volLabel
	reqObj.Message.Recipients = append(reqObj.Message.Recipients, recipientObj)

	err := ml.sendRequest(reqObj, recipientObj)
	if err != nil {
		return err
	}
	return nil
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

func (mailerStub) SendVolumeCreated(userID, label string) error {
	logrus.Infof("Mailer.SendVolumeCreated userID=%s label=%s", userID, label)
	return nil
}

func (mailerStub) SendVolumeDeleted(userID, label string) error {
	logrus.Infof("Mailer.SendVolumeDeleted userID=%s label=%s", userID, label)
	return nil
}
