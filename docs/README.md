# Bindplane Log Agent Documentation

This repo contains documentation for the Bindplane Log Agent.

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

## What plugins are available?

Input plugins:
- [File input](/plugins/file_input.md)
- [TCP input](/plugins/tcp_input.md)
- [UDP input](/plugins/udp_input.md)
- [Journald input](/plugins/journald_input.md)
- [Generate input](/plugins/generate_input.md)

Parser plugins:
- [JSON parser](/plugins/json_parser.md)
- [Regex parser](/plugins/regex_parser.md)
- [Syslog parser](/plugins/syslog_parser.md)

Output plugins:
- [Google Cloud Logging](/plugins/google_cloud_output.md)
- [Elasticsearch](/plugins/elastic_output.md)
- [Stdout](/plugins/stdout.md)

General purpose plugins:
- [Restructure records](/plugins/restructure.md)
- [Copy to multiple outputs](/plugins/copy.md)
- [Router](/plugins/router.md)
- [Metadata](/plugins/metadata.md)

Or take a look at our [Custom Plugins](/custom_plugins.md) for a technology-specific, drop-in log parsing.
