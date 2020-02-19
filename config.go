package log

type Config struct {
	LogLevel string
	Plugins  []PluginConfig
}

type PluginConfig interface{}
