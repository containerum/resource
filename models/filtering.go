package models

import "reflect"

var nsFilterMap = make(map[string]int)
var volFilterMap = make(map[string]int)

func mapFields(tp reflect.Type, mp map[string]int) {
	if tp.Kind() != reflect.Struct {
		panic("struct mapping only supported")
	}
	for i := 0; i < tp.NumField(); i++ {
		if tag, hasTag := tp.Field(i).Tag.Lookup("filter"); hasTag {
			mp[tag] = i
		}
	}
}

func init() {
	mapFields(reflect.TypeOf(NamespaceFilterParams{}), nsFilterMap)
	mapFields(reflect.TypeOf(VolumeFilterParams{}), volFilterMap)
}

// NamespaceFilterParams is a special struct to apply sql filters to in namespaces retrieving requests.
type NamespaceFilterParams struct {
	NotDeleted bool `filter:"not_deleted"` // include only namespaces which marked as not deleted
	Deleted    bool `filter:"deleted"`     // include only namespaces which marked as deleted
	Limited    bool `filter:"limited"`     // include only namespaces which marked as limited
	NotLimited bool `filter:"not_limited"` // include only namespaces which marked as not limited
	Owners     bool `filter:"owned"`       // include only namespaces which user owns
}

// ParseNamespaceFilterParams parses a string filters
func ParseNamespaceFilterParams(filters ...string) (ret NamespaceFilterParams) {
	value := reflect.ValueOf(&ret)
	for _, filter := range filters {
		if field, hasField := nsFilterMap[filter]; hasField {
			value.Field(field).SetBool(true)
		}
	}
	return
}

// VolumeFilterParams is a special struct to apply sql filters to in volumes retrieving requests.
type VolumeFilterParams struct {
	NotDeleted    bool `filter:"not_deleted"`    // include only volumes which marked as not deleted
	Deleted       bool `filter:"deleted"`        // include only volumes which marked as deleted
	Limited       bool `filter:"limited"`        // include only volumes which marked as limited
	NotLimited    bool `filter:"not_limited"`    // include only volumes which marked as not limited
	Owners        bool `filter:"owned"`          // include only volumes which user owns
	Persistent    bool `filter:"persistent"`     // include only persistent volumes
	NotPersistent bool `filter:"not_persistent"` // include only non-persistent volumes
}

// ParseVolumeFilterParams parses a string filters
func ParseVolumeFilterParams(filters ...string) (ret VolumeFilterParams) {
	value := reflect.ValueOf(&ret)
	for _, filter := range filters {
		if field, hasField := volFilterMap[filter]; hasField {
			value.Field(field).SetBool(true)
		}
	}
	return
}
