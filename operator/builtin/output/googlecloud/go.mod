module github.com/observiq/stanza/operator/builtin/output/googlecloud

go 1.14

require (
	cloud.google.com/go/logging v1.1.0
	github.com/golang/protobuf v1.4.2
	github.com/observiq/stanza v0.9.14
	github.com/stretchr/testify v1.6.1
	go.uber.org/zap v1.15.0
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	google.golang.org/api v0.30.0
	google.golang.org/genproto v0.0.0-20200827165113-ac2560b5e952
	google.golang.org/grpc v1.31.0
)

replace github.com/observiq/stanza => ../../../../
