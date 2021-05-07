module github.com/observiq/stanza/operator/builtin/input/k8sevent

go 1.14

require (
	github.com/observiq/stanza v0.13.10
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.16.0
	k8s.io/api v0.19.0
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v0.19.0
)

replace github.com/observiq/stanza => ../../../../
