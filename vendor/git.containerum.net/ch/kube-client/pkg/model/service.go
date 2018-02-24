package model

type Protocol string
type ServiceType string

const (
	UDP Protocol = "UDP"
	TCP Protocol = "TCP"
)

type Service struct {
	Name      string        `json:"name" binding:"required"`
	CreatedAt *int64        `json:"created_at,omitempty"`
	Deploy    string        `json:"deploy,omitempty"`
	IPs       []string      `json:"ips,omitempty"`
	Domain    string        `json:"domain,omitempty"`
	Ports     []ServicePort `json:"ports" binding:"required,dive"`
}

type ServicePort struct {
	Name       string   `json:"name" binding:"required"`
	Port       int      `json:"port" binding:"required"`
	TargetPort *int     `json:"target_port,omitempty"`
	Protocol   Protocol `json:"protocol" binding:"required"`
}
