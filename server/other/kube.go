package other

import (
	"encoding/json"
	"io/ioutil"
	"bytes"
	"context"
	"net/http"
	"net/url"
	"fmt"
)

type Kube interface {
	CreateNamespace(ctx context.Context, name string, cpu, memory uint) error
	DeleteNamespace(ctx context.Context, name string) error
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

func (kub kube) CreateNamespace(ctx context.Context, name string, cpu, memory uint) error {
	refURL := &url.URL{
		Path:     "/api/v1/namespaces",
		RawQuery: fmt.Sprintf("cpu=%d&memory=%d", cpu, memory),
	}
	var ns = map[string]interface{}{
		"kind": "Namespace",
		"apiVersion": "v1",
		"metadata": map[string]interface{}{
			"name": name,
		},
	}
	nsBytes, _ := json.Marshal(ns)
	nsBuffer := bytes.NewReader(nsBytes)
	req, _ := http.NewRequest("POST", kub.u.ResolveReference(refURL).String(), nsBuffer)
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
		Path:     "/api/v1/namespaces",
	}
	req, _ := http.NewRequest("DELETE", kub.u.ResolveReference(refURL).String(), nil)
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
