# Stanza Log Agent Documentation

This repo contains documentation for the Stanza Log Agent.

## How do I configure the agent?
The agent is configured using a YAML config file that is passed in using the `--config` flag. This file defines a collection of operators beneath a top-level `pipeline` key. Each operator possesses a `type` and `id` field.

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
An operator is the most basic unit of log processing. Each operator fulfills only a single responsibility, such as reading lines from a file, or parsing JSON from a field. These operators are then chained together in a pipeline to achieve a desired result.

For instance, a user may read lines from a file using the `file_input` operator. From there, the results of this operation may be sent to a `regex_parser` operator that creates fields based on a regex pattern. And then finally, these results may be sent to a `elastic_output` operator that writes each line to Elasticsearch.

## What operators are available?

Inputs:
- [File input](/docs/operators/file_input.md)
- [Windows Event Log input](/docs/operators/windows_eventlog_input.md)
- [TCP input](/docs/operators/tcp_input.md)
- [UDP input](/docs/operators/udp_input.md)
- [Journald input](/docs/operators/journald_input.md)
- [Generate input](/docs/operators/generate_input.md)

Parsers:
- [JSON parser](/docs/operators/json_parser.md)
- [Regex parser](/docs/operators/regex_parser.md)
- [Syslog parser](/docs/operators/syslog_parser.md)
- [Severity parser](/docs/operators/severity_parser.md)
- [Time parser](/docs/operators/time_parser.md)

Outputs:
- [Google Cloud Logging](/docs/operators/google_cloud_output.md)
- [Elasticsearch](/docs/operators/elastic_output.md)
- [Stdout](/docs/operators/stdout.md)

General purpose:
- [Metadata](/docs/operators/metadata.md)
- [Restructure records](/docs/operators/restructure.md)
- [Router](/docs/operators/router.md)
- [Filter](/docs/operators/filter.md)
- [Kubernetes Metadata Decorator](/docs/operators/k8s_metadata_decorator.md)
- [Host Metadata](/docs/operators/host_metadata.md)
- [Rate limit](/docs/operators/rate_limit.md)

Or create your own [plugins](/docs/plugins.md) for a technology-specific use case.
