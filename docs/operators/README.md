## What is an operator?
An operator is the most basic unit of log processing. Each operator fulfills a single responsibility, such as reading lines from a file, or parsing JSON from a field. Operators are then chained together in a pipeline to achieve a desired result.

For instance, a user may read lines from a file using the `file_input` operator. From there, the results of this operation may be sent to a `regex_parser` operator that creates fields based on a regex pattern. And then finally, these results may be sent to a `elastic_output` operator that writes each line to Elasticsearch.


## What operators are available?

Inputs:
- [File](/docs/operators/file_input.md)
- [Windows Event Log](/docs/operators/windows_eventlog_input.md)
- [TCP](/docs/operators/tcp_input.md)
- [UDP](/docs/operators/udp_input.md)
- [Journald](/docs/operators/journald_input.md)
- [Generate](/docs/operators/generate_input.md)

Parsers:
- [CSV](/docs/operators/csv_parser.md)
- [JSON](/docs/operators/json_parser.md)
- [Regex](/docs/operators/regex_parser.md)
- [Syslog](/docs/operators/syslog_parser.md)
- [Severity](/docs/operators/severity_parser.md)
- [Time](/docs/operators/time_parser.md)

Outputs:
- [Google Cloud Logging](/docs/operators/google_cloud_output.md)
- [Elasticsearch](/docs/operators/elastic_output.md)
- [Stdout](/docs/operators/stdout.md)
- [File](docs/operators/file_output.md)

General purpose:
- [Rate Limit](/docs/operators/rate_limit.md)
- [Filter](/docs/operators/filter.md)
- [Router](/docs/operators/router.md)
- [Metadata](/docs/operators/metadata.md)
- [Restructure](/docs/operators/restructure.md)
- [Host Metadata](/docs/operators/host_metadata.md)
- [Kubernetes Metadata Decorator](/docs/operators/k8s_metadata_decorator.md)

Or create your own [plugins](/docs/plugins.md) for a technology-specific use case.
