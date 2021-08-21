module github.com/observiq/stanza

go 1.14

require (
	cloud.google.com/go/logging v1.4.2
	github.com/Azure/azure-event-hubs-go/v3 v3.3.11
	github.com/antonmedv/expr v1.9.0
	github.com/aws/aws-sdk-go v1.40.26
	github.com/bmatcuk/doublestar/v2 v2.0.4
	github.com/cenkalti/backoff/v4 v4.1.1
	github.com/elastic/go-elasticsearch/v7 v7.13.0
	github.com/golang/protobuf v1.5.2
	github.com/hashicorp/go-uuid v1.0.2
	github.com/jpillora/backoff v1.0.0
	github.com/json-iterator/go v1.1.11
	github.com/kardianos/service v1.2.0
	github.com/mitchellh/mapstructure v1.4.1
	github.com/observiq/ctimefmt v1.0.0
	github.com/observiq/go-syslog/v3 v3.0.2
	github.com/observiq/goflow/v3 v3.4.4
	github.com/observiq/nanojack v0.0.0-20201106172433-343928847ebc
	github.com/spf13/cobra v1.1.3
	github.com/stretchr/testify v1.7.0
	github.com/testcontainers/testcontainers-go v0.11.1
	go.etcd.io/bbolt v1.3.6
	go.opentelemetry.io/collector v0.13.0
	go.uber.org/multierr v1.7.0
	go.uber.org/zap v1.16.0
	golang.org/x/oauth2 v0.0.0-20210628180205-a41e5a781914
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c
	golang.org/x/text v0.3.7
	gonum.org/v1/gonum v0.9.1
	google.golang.org/api v0.52.0
	google.golang.org/genproto v0.0.0-20210722135532-667f2b7c528f
	google.golang.org/grpc v1.40.0
	gopkg.in/yaml.v2 v2.4.0
	// k8s.io modules should be the same version
	k8s.io/api v0.22.0
	k8s.io/apimachinery v0.22.0
	k8s.io/client-go v0.22.0
)
