package billing

import "git.containerum.net/ch/json-types/resource-service"

type GetNamespaceTariffResponse = NamespaceTariff

type GetVolumeTariffResponse = VolumeTariff

type SubscribeTariffRequest struct {
	TariffID      string        `json:"tariff_id"`
	ResourceType  resource.Kind `json:"resource_type"`
	ResourceLabel string        `json:"resource_label"`
	ResourceID    string        `json:"resource_id"`
}

type UnsubscribeTariffRequest struct {
	ResourceID string `json:"resource_id"`
}
