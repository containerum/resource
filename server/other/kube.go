package other

type Kube interface {
	CreateNamespace(name string, cpu, memory uint) error
	DeleteNamespace(name string) error
}

type kube struct {
	c *http.Client
	u *url.URL
}

func (kube kube) CreateNamespace(name string, cpu, memory uint) error {
}

func (kube kube) DeleteNamespace(name string) error {
}
