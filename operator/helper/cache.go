package helper

type CacheConfig struct {
	CacheType    string `json:"type" yaml:"type"`
	CacheMaxSize uint   `json:"size" yaml:"size"`
}

func NewCacheConfig() CacheConfig {
	return CacheConfig{}
}
