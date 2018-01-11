package models

import (
	"time"
)

// Internally used models.

type AccessLevel string // constants AOwner, etc.

const (
	AOwner      AccessLevel = "owner"
	AWrite      AccessLevel = "write"
	AReadDelete AccessLevel = "readdelete"
	ARead       AccessLevel = "read"
	ANone       AccessLevel = "none"
)

type accessRecord struct {
	UserID           string      `json:"user_id,omitempty"`
	Access           AccessLevel `json:"access_level,omitempty"`
	Limited          bool        `json:"limited,omitempty"`
	NewAccess        AccessLevel `json:"new_access_level,omitempty"`
	AccessChangeTime time.Time   `json:"access_level_change_time,omitempty"`
}
