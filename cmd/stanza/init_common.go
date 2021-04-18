package main

import (
	// Load packages when importing input operators
	_ "github.com/observiq/stanza/operator/builtin/input/azure/eventhub"
	_ "github.com/observiq/stanza/operator/builtin/input/file"
	_ "github.com/observiq/stanza/operator/builtin/input/forward"
	_ "github.com/observiq/stanza/operator/builtin/input/generate"
	_ "github.com/observiq/stanza/operator/builtin/input/k8sevent"
	_ "github.com/observiq/stanza/operator/builtin/input/stanza"
	_ "github.com/observiq/stanza/operator/builtin/input/stdin"
	_ "github.com/observiq/stanza/operator/builtin/input/tcp"
	_ "github.com/observiq/stanza/operator/builtin/input/udp"

	_ "github.com/observiq/stanza/operator/builtin/parser/csv"
	_ "github.com/observiq/stanza/operator/builtin/parser/json"
	_ "github.com/observiq/stanza/operator/builtin/parser/regex"
	_ "github.com/observiq/stanza/operator/builtin/parser/severity"
	_ "github.com/observiq/stanza/operator/builtin/parser/syslog"
	_ "github.com/observiq/stanza/operator/builtin/parser/time"
	_ "github.com/observiq/stanza/operator/builtin/parser/uri"

	_ "github.com/observiq/stanza/operator/builtin/transformer/filter"
	_ "github.com/observiq/stanza/operator/builtin/transformer/hostmetadata"
	_ "github.com/observiq/stanza/operator/builtin/transformer/k8smetadata"
	_ "github.com/observiq/stanza/operator/builtin/transformer/metadata"
	_ "github.com/observiq/stanza/operator/builtin/transformer/noop"
	_ "github.com/observiq/stanza/operator/builtin/transformer/ratelimit"
	_ "github.com/observiq/stanza/operator/builtin/transformer/recombine"
	_ "github.com/observiq/stanza/operator/builtin/transformer/restructure"
	_ "github.com/observiq/stanza/operator/builtin/transformer/router"

	_ "github.com/observiq/stanza/operator/builtin/output/drop"
	_ "github.com/observiq/stanza/operator/builtin/output/elastic"
	_ "github.com/observiq/stanza/operator/builtin/output/file"
	_ "github.com/observiq/stanza/operator/builtin/output/forward"
	_ "github.com/observiq/stanza/operator/builtin/output/googlecloud"
	_ "github.com/observiq/stanza/operator/builtin/output/newrelic"
	_ "github.com/observiq/stanza/operator/builtin/output/otlp"
	_ "github.com/observiq/stanza/operator/builtin/output/stdout"
)
