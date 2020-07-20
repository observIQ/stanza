# Carbon Log Agent Documentation

This repo contains documentation for the Carbon Log Agent.

## How do I configure the agent?
The agent is configured using a YAML config file that is passed in using the `--config` flag. This file defines a collection of plugins beneath a top-level `plugins` key. Each plugin possesses a `type` and `id` field.

```yaml
plugins:
  - type: udp_input
    listen_address: :5141

  - type: syslog_parser
    parse_from: message
    protocol: rfc5424

  - type: elastic_output
```

## What is an operator?
An operator is the most basic unit of log processing. Each operator fulfills only a single responsibility, such as reading lines from a file, or parsing JSON from a field. These plugins are then chained together in a pipeline to achieve a desired result.

For instance, a user may read lines from a file using the `file_input` plugin. From there, the results of this operation may be sent to a `regex_parser` plugin that creates fields based on a regex pattern. And then finally, these results may be sent to a `elastic_output` plugin that writes each line to Elasticsearch.

## What operators are available?

Inputs:
- [File input](/docs/plugins/file_input.md)
- [TCP input](/docs/plugins/tcp_input.md)
- [UDP input](/docs/plugins/udp_input.md)
- [Journald input](/docs/plugins/journald_input.md)
- [Generate input](/docs/plugins/generate_input.md)

Parsers:
- [JSON parser](/docs/plugins/json_parser.md)
- [Regex parser](/docs/plugins/regex_parser.md)
- [Syslog parser](/docs/plugins/syslog_parser.md)
- [Severity parser](/docs/plugins/severity_parser.md)
- [Time parser](/docs/plugins/time_parser.md)

Outputs:
- [Google Cloud Logging](/docs/plugins/google_cloud_output.md)
- [Elasticsearch](/docs/plugins/elastic_output.md)
- [Stdout](/docs/plugins/stdout.md)

General purpose:
- [Metadata](/docs/plugins/metadata.md)
- [Restructure records](/docs/plugins/restructure.md)
- [Router](/docs/plugins/router.md)
- [Kubernetes Metadata Decorator](/docs/plugins/k8s_metadata_decorator.md)
- [Rate limit](/docs/plugins/rate_limit.md)

Or create your own [plugins](/docs/plugins.md) for a technology-specific use case.
