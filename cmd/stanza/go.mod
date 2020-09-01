module github.com/observiq/stanza/cmd/stanza

go 1.14

require (
	github.com/elastic/go-elasticsearch/v7 v7.9.0 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/kardianos/service v1.1.0
	github.com/observiq/go-syslog/v3 v3.0.2 // indirect
	github.com/observiq/stanza v0.9.14
	github.com/observiq/stanza/operator/builtin/input/file v0.0.0-00010101000000-000000000000
	github.com/observiq/stanza/operator/builtin/input/journald v0.0.0-00010101000000-000000000000
	github.com/observiq/stanza/operator/builtin/input/k8sevent v0.0.0-00010101000000-000000000000
	github.com/observiq/stanza/operator/builtin/input/tcp v0.0.0-00010101000000-000000000000
	github.com/observiq/stanza/operator/builtin/input/udp v0.0.0-00010101000000-000000000000
	github.com/observiq/stanza/operator/builtin/output/elastic v0.0.0-00010101000000-000000000000
	github.com/observiq/stanza/operator/builtin/output/file v0.0.0-00010101000000-000000000000
	github.com/observiq/stanza/operator/builtin/output/googlecloud v0.0.0-00010101000000-000000000000
	github.com/observiq/stanza/operator/builtin/output/stdout v0.0.0-00010101000000-000000000000
	github.com/observiq/stanza/operator/builtin/parser/json v0.0.0-00010101000000-000000000000
	github.com/observiq/stanza/operator/builtin/parser/regex v0.0.0-00010101000000-000000000000
	github.com/observiq/stanza/operator/builtin/parser/severity v0.0.0-00010101000000-000000000000
	github.com/observiq/stanza/operator/builtin/parser/syslog v0.0.0-00010101000000-000000000000
	github.com/observiq/stanza/operator/builtin/parser/time v0.0.0-00010101000000-000000000000
	github.com/observiq/stanza/operator/builtin/transformer/filter v0.0.0-00010101000000-000000000000
	github.com/observiq/stanza/operator/builtin/transformer/hostmetadata v0.0.0-00010101000000-000000000000
	github.com/observiq/stanza/operator/builtin/transformer/k8smetadata v0.0.0-00010101000000-000000000000
	github.com/observiq/stanza/operator/builtin/transformer/metadata v0.0.0-00010101000000-000000000000
	github.com/observiq/stanza/operator/builtin/transformer/ratelimit v0.0.0-00010101000000-000000000000
	github.com/observiq/stanza/operator/builtin/transformer/restructure v0.0.0-00010101000000-000000000000
	github.com/observiq/stanza/operator/builtin/transformer/router v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.0.0
	github.com/stretchr/testify v1.6.1
	go.etcd.io/bbolt v1.3.5
	go.uber.org/zap v1.15.0
	k8s.io/api v0.19.0 // indirect
	k8s.io/klog v1.0.0 // indirect
	k8s.io/utils v0.0.0-20200821003339-5e75c0163111 // indirect
)

replace github.com/observiq/stanza => ../../

replace github.com/observiq/stanza/operator/builtin/input/file => ../../operator/builtin/input/file

replace github.com/observiq/stanza/operator/builtin/input/generate => ../../operator/builtin/input/generate

replace github.com/observiq/stanza/operator/builtin/input/journald => ../../operator/builtin/input/journald

replace github.com/observiq/stanza/operator/builtin/input/k8sevent => ../../operator/builtin/input/k8sevent

replace github.com/observiq/stanza/operator/builtin/input/tcp => ../../operator/builtin/input/tcp

replace github.com/observiq/stanza/operator/builtin/input/udp => ../../operator/builtin/input/udp

replace github.com/observiq/stanza/operator/builtin/parser/json => ../../operator/builtin/parser/json

replace github.com/observiq/stanza/operator/builtin/parser/regex => ../../operator/builtin/parser/regex

replace github.com/observiq/stanza/operator/builtin/parser/severity => ../../operator/builtin/parser/severity

replace github.com/observiq/stanza/operator/builtin/parser/syslog => ../../operator/builtin/parser/syslog

replace github.com/observiq/stanza/operator/builtin/parser/time => ../../operator/builtin/parser/time

replace github.com/observiq/stanza/operator/builtin/transformer/filter => ../../operator/builtin/transformer/filter

replace github.com/observiq/stanza/operator/builtin/transformer/hostmetadata => ../../operator/builtin/transformer/hostmetadata

replace github.com/observiq/stanza/operator/builtin/transformer/k8smetadata => ../../operator/builtin/transformer/k8smetadata

replace github.com/observiq/stanza/operator/builtin/transformer/metadata => ../../operator/builtin/transformer/metadata

replace github.com/observiq/stanza/operator/builtin/transformer/noop => ../../operator/builtin/transformer/noop

replace github.com/observiq/stanza/operator/builtin/transformer/ratelimit => ../../operator/builtin/transformer/ratelimit

replace github.com/observiq/stanza/operator/builtin/transformer/restructure => ../../operator/builtin/transformer/restructure

replace github.com/observiq/stanza/operator/builtin/transformer/router => ../../operator/builtin/transformer/router

replace github.com/observiq/stanza/operator/builtin/output/drop => ../../operator/builtin/output/drop

replace github.com/observiq/stanza/operator/builtin/output/elastic => ../../operator/builtin/output/elastic

replace github.com/observiq/stanza/operator/builtin/output/file => ../../operator/builtin/output/file

replace github.com/observiq/stanza/operator/builtin/output/googlecloud => ../../operator/builtin/output/googlecloud

replace github.com/observiq/stanza/operator/builtin/output/stdout => ../../operator/builtin/output/stdout
