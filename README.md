![Go](https://github.com/bluemedora/bplogagent/workflows/Go/badge.svg)

# Bindplane Log Agent

## Installation

If you have a `go` environment set up, just `go get github.com/bluemedora/bplogagent`. Just make sure that `$GOPATH/bin` is in your `$PATH`.

## How do I run the agent?
- Run: `bplogagent --config {config_file_location}`
- Test: `go test -cover -race ./...` (cover and race optional)

## How do I configure the agent?
The agent is configured using a YAML config file that is passed in using the `--config` flag. This file defines a collection of plugins beneath a top-level `plugins` key. Each plugin possesses a `type` and `id` field.

```yaml
plugins:
  - id: plugin_one
    type: udp_input
    listen_address: :5141
    output: plugin_two

  - id: plugin_two
    type: syslog_parser
    parse_from: message
    protocol: rfc5424
    output: plugin_three

  - id: plugin_three
    type: elastic_output
```

## What is a plugin?
A plugin is the most basic unit of log monitoring. Each plugin fulfills only a single responsibility, such as reading lines from a file, or parsing JSON from a field. These plugins are then chained together in a pipeline to achieve a desired result.

For instance, a user may read lines from a file using the `file_input` plugin. From there, the results of this operation may be sent to a `regex_parser` plugin that creates fields based on a regex pattern. And then finally, these results may be sent to a `file_output` plugin that writes lines to a file.

## Where are plugins located?
All plugins are built with the agent. They are located within the [builtin](plugin/builtin) package.

In terms of style, the name of each plugin file should be formatted as `{category}_{responsibility}.go`. By prefacing the file name with a category, plugins that share common traits can be grouped alphabetically.

Example categories include:
- `input` (for plugins that discover logs from an external location)
- `output` (for plugins that send logs to an external location)
- `parser` (for plugins that transform logs)
- `filter` (for plugins that alter a log's path in the pipeline)
- `plugin` (for plugins that don't fit into a category)

## How do I build a plugin?
In order to build a plugin, follow these three steps:
1. Build a unique plugin struct that satisfies the [plugin](plugin/plugin.go) interface. This struct will define what your plugin does when executed in the pipeline.

```go
type ExamplePlugin struct {
	FilePath string
}

func (p *ExamplePlugin) Process(entry *entry.Entry) error {
	// Processing logic
}
```

2. Build a unique config struct that satisfies the [config](plugin/config.go) interface. This struct will define the parameters used to configure and build your plugin struct in step 1.

```go
type ExamplePluginConfig struct {
	filePath string
}

func (c ExamplePluginConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	return &ExamplePlugin{
		filePath: c.FilePath,
	}, nil
}
```

3. Register your config struct in the plugin registry using an init hook. This will ensure that the agent knows about your plugin at runtime and can build it from a  YAML config.

```go
func init() {
	plugin.Register("example_plugin", &ExamplePluginConfig{})
}
```

## Any tips for building plugins?
We highly recommend that developers take advantage of [helpers](plugin/helper) when building their plugins. Helpers are structs that help satisfy common behavior shared across many plugins. By embedding these structs, you can skip having to satisfy certain aspects of the `plugin` and `config` interfaces.

For example, almost all plugins should embed the [BasicPlugin](plugin/helper/basic_plugin.go) helper, as it provides simple functionality for returning a plugin id and plugin type.

```go
// ExamplePlugin is a basic plugin, with a basic lifecycle, that consumes
// but doesn't send log entries. Rather than implementing every part of the plugin
// interface, we can embed the following helpers to achieve this effect.
type ExamplePlugin struct {
	helper.BasicPlugin
	helper.BasicLifecycle
	helper.BasicOutput
}
```
