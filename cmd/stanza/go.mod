module github.com/observiq/stanza/cmd/stanza

go 1.14

require (
	cloud.google.com/go/logging v1.0.0 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/kardianos/service v1.1.0
	github.com/observiq/go-syslog/v3 v3.0.2 // indirect
	github.com/observiq/stanza v0.9.12
	github.com/observiq/stanza/operator/builtin v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.0.0
	go.etcd.io/bbolt v1.3.5
	go.uber.org/zap v1.15.0
)

replace github.com/observiq/stanza => ../../

replace github.com/observiq/stanza/operator/builtin => ../../operator/builtin
