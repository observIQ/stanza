[![<observIQ>](https://circleci.com/gh/observIQ/observiq-logagent.svg?style=shield&circle-token=980a514f9dc5a48ac2b8e61a4cdb7555ea5646ca)](https://app.circleci.com/pipelines/github/observIQ/observiq-logagent)
[![codecov](https://codecov.io/gh/observIQ/observiq-logagent/branch/master/graph/badge.svg?token=i50h3UFXLg)](https://codecov.io/gh/observIQ/observiq-logagent)

# observIQ Log Agent

## How do I install the agent?
#### Golang Project
```shell
go get github.com/observiq/bplogagent
```
#### Linux
```shell
sh -c "$(curl -fsSl https://github.com/observiq/observiq-logagent/releases/latest/download/unix-install.sh)" unix-install.sh
```
#### Darwin
```shell
sh -c "$(curl -fsSl https://github.com/observiq/observiq-logagent/releases/latest/download/unix-install.sh)" unix-install.sh
```
#### Windows
```pwsh
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12 ; Invoke-Expression ((New-Object net.webclient).DownloadString('https://github.com/observiq/observiq-logagent/releases/latest/download/windows-install.ps1')); Log-Agent-Install
```

## How do I run the agent?
#### Manual
```shell
# Example Command
bplogagent --config ./config.yaml --plugin_dir ./plugins --database ./offsets.db

# Supported flags:
--config      The location of the agent config file
--plugin_dir  The location of the custom plugins directory
--database    The location of the offsets database file
--log_file    The location of the agent log file
--debug       Enables debug logging
```
#### Linux
```shell
# systemd
systemctl start bplogagent

# sysv
service bplogagent start
```
#### Darwin
```shell
launchctl start com.observiq.bplogagent
```
#### Windows
```pwsh
Start-Service -Name "bplogagent"
```

## How do I configure the agent?
The agent is configured using a YAML config file that is passed in using the `--config` flag. This file defines a collection of plugins beneath a top-level `pipeline` key. Each plugin possesses a `type` and `id` field.

```yaml
pipeline:
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

If you have [`graphviz`](https://graphviz.org/) installed, you can view the plugin graph with a command like:
```bash
bplogagent graph --config './config.yaml' | dot -Tsvg -o /tmp/graph.svg && open /tmp/graph.svg
```

## What is a plugin?
A plugin is the most basic unit of log monitoring. Each plugin fulfills a single responsibility, such as reading lines from a file, or parsing JSON from a field. These plugins are then chained together in a pipeline to achieve a desired result.

For instance, a user may read lines from a file using the `file_input` plugin. From there, the results of this operation may be sent to a `regex_parser` plugin that creates fields based on a regex pattern. And then finally, these results may be sent to a `file_output` plugin that writes lines to a file.

## What plugins are available?
For more information on what plugins are available and how to configure them, take a look at our [documentation](/docs/README.md).

## Can I route logs through a proxy server?
Yes. The agent will respect `http_proxy` and `https_proxy` environment variables, as defined in Golang's [net/http](https://golang.org/pkg/net/http/#ProxyFromEnvironment) package.

## How do I contribute?
Take a look at our contribution guidelines in [`CONTRIBUTING.md`](./CONTRIBUTING.md)
