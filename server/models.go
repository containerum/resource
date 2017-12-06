package server

import (
	"github.com/satori/go.uuid"
	"time"
)

// Internally used models.

type Namespace struct {
	ID            *uuid.UUID `json:"id,omitempty"`
	CreateTime    *time.Time `json:"create_time,omitempty"`
	Label         *string    `json:"label,omitempty"`
	RAM           *int       `json:"ram,omitempty"`
	CPU           *int       `json:"cpu,omitempty"`
	MaxExtService *int       `json:"max_ext_service,omitempty"`
	MaxIntService *int       `json:"max_int_service,omitempty"`
	MaxTraffic    *int       `json:"max_traffic,omitempty"`
	Deleted       *bool      `json:"deleted,omitempty"`
	DeleteTime    *time.Time `json:"delete_time,omitempty"`
	TariffID      *uuid.UUID `json:"tariff_id,omitempty"`

	Volumes []Volume `json:"volumes,omitempty"`
}

type Volume struct {
	ID         *uuid.UUID `json:"id,omitempty"`
	CreateTime *time.Time `json:"create_time,omitempty"`
	Label      *string    `json:"label,omitempty"`
	Storage    *int       `json:"storage,omitempty"`
	Replicas   *int       `json:"replicas,omitempty"`
	Persistent *bool      `json:"persistent,omitempty"`
	Deleted    *bool      `json:"deleted,omitempty"`
	DeleteTime *time.Time `json:"delete_time,omitempty"`
	TariffID   *uuid.UUID `json:"tariff_id,omitempty"`
}

type Service struct {
}
