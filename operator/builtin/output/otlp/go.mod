module github.com/observiq/stanza/operator/builtin/output/otlp

go 1.14

require (
	github.com/mitchellh/mapstructure v1.3.3
	github.com/observiq/stanza v0.12.5
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/collector v0.12.0
	google.golang.org/grpc v1.32.0
	gopkg.in/yaml.v1 v1.0.0-20140924161607-9f9df34309c0
	gopkg.in/yaml.v2 v2.3.0
)

replace github.com/observiq/stanza => ../../../../
