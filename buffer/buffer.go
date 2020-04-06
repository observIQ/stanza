package buffer

import "github.com/bluemedora/bplogagent/entry"

type BaseBuffer struct {
	// TODO enforce these
	bufferedByteLimit  int
	bufferedCountLimit int

	bundleByteThreshold  int
	bundleCountThreshold int
	flushDelayThreshold  float64

	bundleByteLimit  int
	bundleCountLimit int

	handler func([]*entry.Entry) error

	// TODO retry behavior
	// TODO customizable overflow action?
	// TODO allow compression?
}
