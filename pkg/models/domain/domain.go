package domain

type Domain struct {
	ID          string   `json:"_id,omitempty"`
	Domain      string   `json:"domain"`
	DomainGroup string   `json:"domain_group"`
	IP          []string `json:"ip"`
}
