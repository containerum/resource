package model

import (
	"github.com/satori/go.uuid"
	"math/big"
	"time"
)

type User struct {
	ID        *string    `json:"user_id,omitempty"`
	Login     *string    `json:"login,omitempty"`
	Country   *int       `json:"country,omitempty"`
	Balance   *big.Rat   `json:"balance,omitempty"`
	BillingID *string    `json:"billing_id,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

type Tariff struct {
	ID        *string  `json:"id,omitempty"`
	Label     *string  `json:"label,omitempty"`
	Type      *string  `json:"type,omitempty"`
	Price     *big.Rat `json:"price,omitempty"`
	IsActive  *bool    `json:"is_active,omitempty"`
	IsPublic  *bool    `json:"is_public,omitempty"`
	BillingID *string  `json:"billing_id,omitempty"`
}

type NamespaceTariff struct {
	ID               *uuid.UUID `json:"id,omitempty"`
	TariffID         *uuid.UUID `json:"tariff_id,omitempty"`
	Description      *string    `json:"description,omitempty"`
	CreatedAt        *time.Time `json:"created_at,omitempty"`

	CpuLimit         *int       `json:"cpu_limit,omitempty"`
	MemoryLimit      *int       `json:"memory_limit,omitempty"`
	Traffic          *int       `json:"traffic,omitempty"`
	TrafficPrice     *big.Rat   `json:"traffic_price,omitempty"`
	ExternalServices *int       `json:"external_services,omitempty"`
	InternalServices *int       `json:"internal_services,omitempty"`

	VV *VolumeTariff `json:"VV,omitempty"`

	IsActive *bool    `json:"is_active,omitempty"`
	IsPublic *bool    `json:"is_public,omitempty"`
	Price    *big.Rat `json:"price,omitempty"`
}

type VolumeTariff struct {
	ID            *uuid.UUID `json:"id,omitempty"`
	TariffID      *uuid.UUID `json:"tariff_id,omitempty"`
	Description   *string    `json:"description,omitempty"`
	CreatedAt     *time.Time `json:"created_at,omitempty"`

	StorageLimit  *int       `json:"storage_limit,omitempty"`
	ReplicasLimit *int       `json:"replicas_limit,omitempty"`
	IsPersistent  *bool      `json:"is_persistent,omitempty"`

	IsActive *bool    `json:"is_active,omitempty"`
	IsPublic *bool    `json:"is_public,omitempty"`
	Price    *big.Rat `json:"price,omitempty"`
}

type Resource struct {
	ResourceID *uuid.UUID `json:"resource_id,omitempty"`
	UserID     *string    `json:"user_id,omitempty"`
	TariffID   *string    `json:"tariff_id,omitempty"`
	BillingID  *string    `json:"billing_id,omitempty"`
	Status     *string    `json:"status,omitempty"`
}
