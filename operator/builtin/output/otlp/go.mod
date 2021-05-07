module github.com/observiq/stanza/operator/builtin/output/otlp

go 1.14

require (
	github.com/mitchellh/mapstructure v1.4.1
	github.com/observiq/stanza v0.12.5
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/collector v0.13.0
	go.uber.org/zap v1.16.0
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/observiq/stanza => ../../../../
