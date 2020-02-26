package plugin

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testInputterExitsOnChannelClose(t *testing.T, inputter Inputter) {
	// Ensure that the plugin output isn't blocked
	if outputter, ok := inputter.(Outputter); ok {
		for _, channel := range outputter.Outputs() {
			consumeEntries(channel)
		}
	}

	// Start the plugin
	wg := new(sync.WaitGroup)
	wg.Add(1)
	err := inputter.Start(wg)
	assert.NoError(t, err)

	// Signal when the plugin exits
	exited := make(chan struct{})
	go func() {
		wg.Wait()
		close(exited)
	}()

	// Ensure the plugin exits quickly when its input channel is closed
	close(inputter.Input())
	select {
	case <-exited:
	case <-time.After(10 * time.Millisecond):
		t.Errorf("Inputter of type %T did not exit in a timely manner when its input channel was closed", inputter)
	}
}

func consumeEntries(channel EntryChannel) {
	go func() {
		_, ok := <-channel
		if !ok {
			return
		}
	}()
}
