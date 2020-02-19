package main

type Config struct {
	LogLevel string
	LogPath  string
	Plugins  []PluginConfig
}

type PluginConfig interface{}
