package cache

// Cache is an interface for caching resources
type Cache interface {
	Get(k interface{}) (v interface{}, cached bool)
	Set(k, v interface{})
}
