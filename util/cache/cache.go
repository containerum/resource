package cache

type Cache interface {
	Get(k interface{}) (v interface{}, cached bool)
	Set(k, v interface{})
}
