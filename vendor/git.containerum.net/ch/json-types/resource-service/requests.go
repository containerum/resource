package resource

type CreateResourceRequest struct {
	TariffID string `json:"tariff-id"`
	Label    string `json:"label"`
}

type RenameResourceRequest struct {
	New string `json:"label"`
}

type SetResourceLockRequest struct {
	Lock *bool `json:"lock"`
}

type SetResourceAccessRequest struct {
	UserID string `json:"user_id"`
	Access string `json:"access"`
}
