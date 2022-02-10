## What is an operator?
An operator is the most basic unit of log processing. Each operator fulfills a single responsibility, such as reading lines from a file, or parsing JSON from a field. Operators are then chained together in a pipeline to achieve a desired result.

For instance, a user may read lines from a file using the `file_input` operator. From there, the results of this operation may be sent to a `regex_parser` operator that creates fields based on a regex pattern. And then finally, these results may be sent to a `elastic_output` operator that writes each line to Elasticsearch.


## What operators are available?

Inputs:
- [AWS Cloudwatch](/docs/operators/aws_cloudwatch_input.md)
- [Azure Event Hub](/docs/operators/azure_event_hub_input.md)
- [Azure Log Analytics](/docs/operators/azure_log_analytics_input.md)
- [File](/docs/operators/file_input.md)
- [Generate](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/generate_input.md)
- [Goflow](/docs/operators/goflow_input.md)
- [HTTP](/docs/operators/http_input.md)
- [Journald](/docs/operators/journald_input.md)
- [Kubernetes Event](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/k8s_event_input.md)
- [Stanza Forward](/docs/operators/forward_input.md)
- [Stanza Self](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/stanza_input.md)
- [Stdin](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/stdin.md)
- [TCP](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/tcp_input.md)
- [UDP](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/udp_input.md)
- [Windows Event Log](/docs/operators/windows_eventlog_input.md)

Parsers:
- [CSV](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/csv_parser.md)
- [JSON](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/json_parser.md)
- [Key Value](/docs/operators/key_value_parser.md)
- [Regex](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/regex_parser.md)
- [Severity](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/severity_parser.md)
- [Syslog](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/syslog_parser.md)
- [Time](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/time_parser.md)
- [URI](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/uri_parser.md)
- [XML](/docs/operators/xml_parser.md)

Outputs:
- [Drop](/docs/operators/drop_output.md)
- [Elasticsearch](/docs/operators/elastic_output.md)
- [File](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/file_output.md)
- [Google Cloud Logging](/docs/operators/google_cloud_output.md)
- [Newrelic](/docs/operators/newrelic_output.md)
- [Stanza Forward](/docs/operators/forward_output.md)
- [Stdout](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/stdout.md)

General purpose:
- [Add](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/add.md)
- [Copy](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/copy.md)
- [Filter](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/filter.md)
- [Flatten](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/flatten.md)
- [Host Metadata](/docs/operators/host_metadata.md)
- [Kubernetes Metadata Decorator](/docs/operators/k8s_metadata_decorator.md)
- [Metadata](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/metadata.md)
- [Move](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/move.md)
- [Recombine](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/recombine.md)
- [Remove](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/remove.md)
- [Rate Limit](/docs/operators/rate_limit.md)
- [Restructure](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/restructure.md)
- [Retain](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/retain.md)
- [Router](https://github.com/open-telemetry/opentelemetry-log-collection/blob/main/docs/operators/router.md)

Or create your own [plugins](/docs/plugins.md) for a technology-specific use case.
