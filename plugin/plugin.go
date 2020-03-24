//go:generate mockery -name=^(Plugin|Inputter|Outputter)$ -output=./testutil -outpkg=testutil -case=snake
package plugin

import (
	"github.com/bluemedora/bplogagent/entry"
)

// Plugin is a log monitoring component with a single responsibility.
type Plugin interface {
	ID() PluginID
	Type() string
	Start() error
	Stop() error
}

// Producer is a plugin that can produce entries to consumers.
type Producer interface {
	Plugin
	Consumers() []Consumer
}

// Consumer is a plugin that can consume entries from producers.
type Consumer interface {
	Plugin
	Consume(*entry.Entry) error
}

type PluginID string
