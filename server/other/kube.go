package other

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/sirupsen/logrus"
)

type Kube interface {
	CreateNamespace(ctx context.Context, name string, cpu, memory uint, label, access string) error
	DeleteNamespace(ctx context.Context, name string) error
}

type kube struct {
	c *http.Client
	u *url.URL
}

func NewKubeHTTP(u *url.URL) Kube {
	k := &kube{
		c: http.DefaultClient,
		u: u,
	}
	return k
}

func (kub kube) CreateNamespace(ctx context.Context, name string, cpu, memory uint, label, access string) error {
	refURL := &url.URL{
		Path:     "api/v1/namespaces",
		RawQuery: fmt.Sprintf("cpu=%d&memory=%d", cpu, memory),
	}
	var ns = map[string]interface{}{
		"kind":       "Namespace",
		"apiVersion": "v1",
		"metadata": map[string]interface{}{
			"name": name,
		},
	}
	nsBytes, _ := json.Marshal(ns)
	nsBuffer := bytes.NewReader(nsBytes)
	req, _ := http.NewRequest("POST", kub.u.ResolveReference(refURL).String(), nsBuffer)
	xUserNamespaceBytes, _ := json.Marshal([]interface{}{
		map[string]string{
			"id":     name,
			"label":  label,
			"access": access,
		},
	})
	xUserNamespaceBase64 := base64.StdEncoding.EncodeToString(xUserNamespaceBytes)
	req.Header.Set("x-user-namespace", xUserNamespaceBase64)
	req = req.WithContext(ctx)
	resp, err := kub.c.Do(req)
	if err != nil {
		return err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("reading response body: %v", err)
	}
	var errObj struct {
		Error string `json:"error"`
	}
	err = json.Unmarshal(respBody, &errObj)
	if err != nil {
		return fmt.Errorf("cannot unmarshal response: %v", err)
	}
	if errObj.Error != "" {
		return fmt.Errorf("kube api error: %s", errObj.Error)
	}
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("kube api error: http status %s", resp.Status)
	}

	return nil
}

func (kub kube) DeleteNamespace(ctx context.Context, name string) error {
	refURL := &url.URL{
		Path: "api/v1/namespaces/"+url.PathEscape(name),
	}
	req, _ := http.NewRequest("DELETE", kub.u.ResolveReference(refURL).String(), nil)
	xUserNamespaceBytes, _ := json.Marshal([]interface{}{
		map[string]string{
			"id":     name,
			"label":  "unknown",
			"access": "owner",
		},
	})
	xUserNamespaceBase64 := base64.StdEncoding.EncodeToString(xUserNamespaceBytes)
	req.Header.Set("x-user-namespace", xUserNamespaceBase64)
	req = req.WithContext(ctx)
	resp, err := kub.c.Do(req)
	if err != nil {
		return fmt.Errorf("http error: %v", err)
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("http error: %v", err)
	}

	if len(respBytes) > 0 {
		var errObj struct {
			Error string `json:"error"`
		}
		err = json.Unmarshal(respBytes, &errObj)
		if err != nil {
			return fmt.Errorf("unmarshal response: %v", err)
		}
		if errObj.Error != "" {
			return fmt.Errorf("kube api error: %s", errObj.Error)
		}
	}
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("kube api error: http status %s", resp.Status)
	}

	return nil
}

func (k kube) String() string {
	return fmt.Sprintf("kube api http client: url=%v", k.u)
}

type kubeStub struct{}

func NewKubeStub() Kube {
	return kubeStub{}
}

func (kubeStub) CreateNamespace(_ context.Context, name string, cpu, memory uint, label, access string) error {
	logrus.Infof("kubeStub.CreateNamespace name=%s cpu=%d memory=%d label=%s access=%s",
		name, cpu, memory, label, access)
	return nil
}

func (kubeStub) DeleteNamespace(_ context.Context, name string) error {
	logrus.Infof("kubeStub.DeleteNamespace name=%s", name)
	return nil
}

func (kubeStub) String() string {
	return "kube api dummy"
}
