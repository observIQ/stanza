[![<BlueMedora>](https://circleci.com/gh/BlueMedora/bplogagent.svg?style=shield&circle-token=b3a927f2797a62157b99f1e592edc0b14b764e8c)](https://app.circleci.com/pipelines/github/BlueMedora/bplogagent)
[![codecov](https://codecov.io/gh/BlueMedora/bplogagent/branch/master/graph/badge.svg?token=MvU9xtiqxd)](https://codecov.io/gh/BlueMedora/bplogagent)

# Bindplane Log Agent

## Installation

If you have a `go` environment set up, just `go get github.com/bluemedora/bplogagent`. Just make sure that `$GOPATH/bin` is in your `$PATH`.

## Documentation

More detailed documentation for the log agent can be found [here](https://github.com/bluemedora/bplogagent-docs).

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

If you have [`graphviz`](https://graphviz.org/) installed, you can view the plugin graph with a command like:
```bash
bplogagent graph --config './config.yaml' | dot -Tsvg -o /tmp/graph.svg && open /tmp/graph.svg
```

## What is a plugin?
A plugin is the most basic unit of log monitoring. Each plugin fulfills only a single responsibility, such as reading lines from a file, or parsing JSON from a field. These plugins are then chained together in a pipeline to achieve a desired result.

For instance, a user may read lines from a file using the `file_input` plugin. From there, the results of this operation may be sent to a `regex_parser` plugin that creates fields based on a regex pattern. And then finally, these results may be sent to a `file_output` plugin that writes lines to a file.

## What plugins are available?
For more information on what plugins are available and how to configure them, take a look at our [plugin documentation](https://github.com/bluemedora/bplogagent-docs).

## How do I contribute?

Take a look at our contribution guidelines in [`CONTRIBUTING.md`](./CONTRIBUTING.md)
