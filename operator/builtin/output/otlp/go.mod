module github.com/observiq/stanza/operator/builtin/output/otlp

go 1.14

require (
	github.com/mitchellh/mapstructure v1.3.2
	github.com/observiq/stanza v0.12.5
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/collector v0.13.0
	go.uber.org/zap v1.16.0
	golang.org/x/text v0.3.5 // indirect
	gopkg.in/yaml.v2 v2.3.0
	github.com/xdg-go/stringprep v1.0.2
)

replace github.com/observiq/stanza => ../../../../
