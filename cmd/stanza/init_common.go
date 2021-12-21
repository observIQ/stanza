package main

import (
	// Load packages when importing input operators
	_ "github.com/observiq/stanza/v2/operator/builtin/input/aws/cloudwatch"
	_ "github.com/observiq/stanza/v2/operator/builtin/input/azure/eventhub"
	_ "github.com/observiq/stanza/v2/operator/builtin/input/azure/loganalytics"
	_ "github.com/observiq/stanza/v2/operator/builtin/input/file"
	_ "github.com/observiq/stanza/v2/operator/builtin/input/forward"
	_ "github.com/observiq/stanza/v2/operator/builtin/input/goflow"
	_ "github.com/observiq/stanza/v2/operator/builtin/input/http"
	_ "github.com/observiq/stanza/v2/operator/builtin/output/elastic"
	_ "github.com/observiq/stanza/v2/operator/builtin/output/forward"
	_ "github.com/observiq/stanza/v2/operator/builtin/output/googlecloud"
	_ "github.com/observiq/stanza/v2/operator/builtin/output/newrelic"
	_ "github.com/observiq/stanza/v2/operator/builtin/parser/keyvalue"
	_ "github.com/observiq/stanza/v2/operator/builtin/parser/xml"
	_ "github.com/observiq/stanza/v2/operator/builtin/transformer/hostmetadata"
	_ "github.com/observiq/stanza/v2/operator/builtin/transformer/k8smetadata"
	_ "github.com/observiq/stanza/v2/operator/builtin/transformer/ratelimit"

	// Otel Operators
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/input/generate"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/input/k8sevent"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/input/stanza"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/input/stdin"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/input/tcp"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/input/udp"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/output/drop"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/output/file"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/output/stdout"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/parser/csv"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/parser/json"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/parser/regex"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/parser/severity"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/parser/syslog"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/parser/time"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/parser/uri"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/add"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/copy"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/filter"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/flatten"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/metadata"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/move"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/noop"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/recombine"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/remove"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/restructure"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/retain"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/router"
)
