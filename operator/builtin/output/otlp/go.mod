module github.com/observiq/stanza/operator/builtin/output/otlp

go 1.14

require (
	github.com/mitchellh/mapstructure v1.3.2
	github.com/observiq/stanza v0.12.5
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/collector v0.12.0
	gopkg.in/yaml.v2 v2.3.0
)

replace github.com/observiq/stanza => ../../../../
