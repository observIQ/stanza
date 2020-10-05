module github.com/observiq/stanza/operator/builtin/input/k8sevent

go 1.14

require (
	github.com/observiq/stanza v0.12.1
	github.com/stretchr/testify v1.6.1
	go.uber.org/zap v1.15.0
	k8s.io/api v0.19.0
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v0.19.0
)

replace github.com/observiq/stanza => ../../../..
