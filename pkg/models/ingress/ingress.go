package ingress

import "github.com/containerum/kube-client/pkg/model"

type Ingress struct {
	model.Ingress
	Owner       string `json:"owner"`
	ID          string `json:"_id"`
	Deleted     bool   `json:"deleted"`
	NamespaceID string `json:"namespaceid"`
}

func (ingr Ingress) Copy() Ingress {
	var cp = ingr
	cp.Rules = append(make([]model.Rule, 0, len(cp.Rules)), cp.Rules...)
	for i, rule := range cp.Rules {
		rule.Path = append(make([]model.Path, 0, len(rule.Path)), rule.Path...)
		cp.Rules[i] = rule
	}
	return cp
}

func (ingr Ingress) Paths() []model.Path {
	var paths = make([]model.Path, 0, len(ingr.Rules))
	for _, rule := range ingr.Rules {
		for _, path := range rule.Path {
			paths = append(paths, path)
		}
	}
	return paths
}
