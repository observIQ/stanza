module github.com/observiq/stanza/cmd/stanza

go 1.14

require (
	github.com/armon/consul-api v0.0.0-20180202201655-eb2c6b5be1b6 // indirect
	github.com/kardianos/service v1.1.0
	github.com/observiq/stanza v0.12.1
	github.com/observiq/stanza/operator/builtin/input/k8sevent v0.1.0
	github.com/observiq/stanza/operator/builtin/input/windows v0.1.1
	github.com/observiq/stanza/operator/builtin/output/elastic v0.1.0
	github.com/observiq/stanza/operator/builtin/output/googlecloud v0.1.0
	github.com/observiq/stanza/operator/builtin/output/newrelic v0.1.0
	github.com/observiq/stanza/operator/builtin/parser/syslog v0.1.0
	github.com/observiq/stanza/operator/builtin/transformer/k8smetadata v0.1.0
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.6.1
	github.com/ugorji/go v1.1.4 // indirect
	github.com/xordataexchange/crypt v0.0.3-0.20170626215501-b2862e3d0a77 // indirect
	go.etcd.io/bbolt v1.3.5
	go.uber.org/zap v1.15.0
)

replace github.com/observiq/stanza => ../../

replace github.com/observiq/stanza/operator/builtin/input/k8sevent => ../../operator/builtin/input/k8sevent

replace github.com/observiq/stanza/operator/builtin/input/windows => ../../operator/builtin/input/windows

replace github.com/observiq/stanza/operator/builtin/parser/syslog => ../../operator/builtin/parser/syslog

replace github.com/observiq/stanza/operator/builtin/transformer/k8smetadata => ../../operator/builtin/transformer/k8smetadata

replace github.com/observiq/stanza/operator/builtin/output/elastic => ../../operator/builtin/output/elastic

replace github.com/observiq/stanza/operator/builtin/output/googlecloud => ../../operator/builtin/output/googlecloud

replace github.com/observiq/stanza/operator/builtin/output/newrelic => ../../operator/builtin/output/newrelic
