package server

import (
	"time"

	"github.com/satori/go.uuid"
)

// Internally used models.

type Namespace struct {
	ID               *uuid.UUID   `json:"id,omitempty"`
	CreateTime       *time.Time   `json:"create_time,omitempty"`
	Deleted          *bool        `json:"deleted,omitempty"`
	DeleteTime       *time.Time   `json:"delete_time,omitempty"`
	UserID           *uuid.UUID   `json:"user_id,omitempty"`
	TariffID         *uuid.UUID   `json:"tariff_id,omitempty"`
	Label            *string      `json:"label,omitempty"`
	Access           *AccessLevel `json:"access,omitempty"`
	AccessChangeTime *time.Time   `json:"access_change_time,omitempty"`
	Limited          *bool        `json:"limited,omitempty"`
	NewAccess        *AccessLevel `json:"new_access,omitempty"`

	RAM           *int `json:"ram,omitempty"`
	CPU           *int `json:"cpu,omitempty"`
	MaxExtService *int `json:"max_ext_service,omitempty"`
	MaxIntService *int `json:"max_int_service,omitempty"`
	MaxTraffic    *int `json:"max_traffic,omitempty"`

	Volumes []Volume `json:"volumes,omitempty"`

	Users []accessRecord `json:"users,omitempty"`
}

type Volume struct {
	ID               *uuid.UUID   `json:"id,omitempty"`
	CreateTime       *time.Time   `json:"create_time,omitempty"`
	Deleted          *bool        `json:"deleted,omitempty"`
	DeleteTime       *time.Time   `json:"delete_time,omitempty"`
	UserID           *uuid.UUID   `json:"user_id,omitempty"`
	TariffID         *uuid.UUID   `json:"tariff_id,omitempty"`
	Label            *string      `json:"label,omitempty"`
	Access           *AccessLevel `json:"access,omitempty"`
	AccessChangeTime *time.Time   `json:"access_change_time,omitempty"`
	Limited          *bool        `json:"limited,omitempty"`
	NewAccess        *AccessLevel `json:"new_access,omitempty"`

	Storage    *int  `json:"storage,omitempty"`
	Replicas   *int  `json:"replicas,omitempty"`
	Persistent *bool `json:"persistent,omitempty"`

	Users []accessRecord `json:"users,omitempty"`
}

type Service struct {
}

type AccessLevel string // constants AOwner, etc.

const (
	AOwner      AccessLevel = "owner"
	AWrite      AccessLevel = "write"
	AReadDelete AccessLevel = "readdelete"
	ARead       AccessLevel = "read"
	ANone       AccessLevel = "none"
)

type accessRecord struct {
	UserID           *uuid.UUID   `json:"user_id,omitempty"`
	Access           *AccessLevel `json:"access_level,omitempty"`
	Limited          *bool        `json:"limited,omitempty"`
	NewAccess        *AccessLevel `json:"new_access_level,omitempty"`
	AccessChangeTime *time.Time   `json:"access_level_change_time,omitempty"`
}
