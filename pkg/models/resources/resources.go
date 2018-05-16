package resources

// Other
type GetResourcesCountResponse struct {
	Deployments int `json:"deployments"`
	ExtServices int `json:"external_services"`
	IntServices int `json:"internal_services"`
	Ingresses   int `json:"ingresses"`
	Pods        int `json:"pods"`
}
