module github.com/observiq/stanza/operator/builtin/output/googlecloud

go 1.14

require (
	cloud.google.com/go/logging v1.1.0
	github.com/golang/protobuf v1.4.2
	github.com/observiq/stanza v0.11.0
	github.com/stretchr/testify v1.6.1
	go.uber.org/zap v1.15.0
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43
	google.golang.org/api v0.31.0
	google.golang.org/genproto v0.0.0-20200831141814-d751682dd103
	google.golang.org/grpc v1.31.1
)

replace github.com/observiq/stanza => ../../../../
