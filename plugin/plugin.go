//go:generate mockery -name=^(Plugin|Producer|Consumer)$ -output=./testutil -outpkg=testutil -case=snake

package plugin

import "github.com/bluemedora/bplogagent/entry"

// Plugin is a log monitoring component with a single responsibility.
type Plugin interface {
	ID() string
	Type() string
	Start() error
	Stop() error
}

// Producer is a plugin that produces entries to consumers.
type Producer interface {
	Plugin
	Consumers() []Consumer
	SetConsumers([]Consumer) error
}

// Consumer is a plugin that consumes entries from producers.
type Consumer interface {
	Plugin
	Consume(*entry.Entry) error
}
