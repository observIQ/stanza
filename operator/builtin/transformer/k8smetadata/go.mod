module github.com/observiq/stanza/operator/builtin/transformer/k8smetadata

go 1.14

require (
	github.com/observiq/stanza v0.12.1
	github.com/stretchr/testify v1.6.1
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v0.19.0
	k8s.io/utils v0.0.0-20200821003339-5e75c0163111 // indirect
)

replace github.com/observiq/stanza => ../../../..
