package cache

type Cache interface {
	Get(key string) (interface{}, bool)
	Add(key string, data interface{})
}
