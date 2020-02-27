package plugin

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
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

	// Close output channels on exit
	if outputter, ok := inputter.(Outputter); ok {
		go func() {
			wg.Wait()
			for _, channel := range outputter.Outputs() {
				close(channel)
			}
		}()
	}

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
		for {
			_, ok := <-channel
			if !ok {
				return
			}
		}
	}()
}

type inputterBenchmark struct {
	fields      int
	depth       int
	fieldLength int
}

func (b inputterBenchmark) String() string {
	return fmt.Sprintf("Fields=%d,Depth=%d,Length=%d", b.fields, b.depth, b.fieldLength)
}

func (b inputterBenchmark) EstimatedBytes() int {
	pow := func(a, b int) int {
		n := 1
		for i := 0; i < b; i++ {
			n = n * a
		}
		return n
	}

	bytes := 0
	for i := 1; i < b.depth+2; i++ {
		bytes += pow(b.fields, i) * b.fieldLength
	}
	bytes += pow(b.fields, b.depth+1) * b.fieldLength

	return bytes
}

var standardInputterBenchmarks = []inputterBenchmark{
	{0, 0, 10},
	{1, 0, 10},
	{1, 0, 100},
	{1, 0, 1000},
	{10, 0, 10},
	{2, 2, 10},
	{2, 10, 10},
}

func benchmarkInputter(b *testing.B, inputter Inputter, bm inputterBenchmark, generate func(int, int, int) map[string]interface{}) {
	if outputter, ok := inputter.(Outputter); ok {
		for _, output := range outputter.Outputs() {
			consumeEntries(output)
		}
	}

	wg := new(sync.WaitGroup)
	wg.Add(1)
	inputter.Start(wg)

	if outputter, ok := inputter.(Outputter); ok {
		go func() {
			wg.Wait()
			for _, output := range outputter.Outputs() {
				close(output)
			}
		}()
	}

	entry := entry.Entry{
		Timestamp: time.Now(),
		Record:    generate(bm.fields, bm.depth, bm.fieldLength),
	}
	encoded, err := json.Marshal(entry)
	assert.NoError(b, err)
	estimatedBytes := int64(len(encoded))
	b.SetBytes(estimatedBytes)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inputter.Input() <- entry
	}

	close(inputter.Input())
}

// generateEntry creates an entry with a configurable number
// of fields per level of the map, as well as a configurable
// number of nested fields for a total of fields ^ depth leaf values
// Example: fields = 1, depth = 2
// {
// 	"asdf1": {
// 		"asdf2": "asdf3",
// 	},
// }
func generateRandomNestedMap(fields int, depth int, bytes int) map[string]interface{} {
	generated := make(map[string]interface{})
	buffer := make([]byte, bytes)
	for i := 0; i < fields; i++ {
		_, _ = rand.Read(buffer)
		field := hex.EncodeToString(buffer)
		if depth == 0 {
			_, _ = rand.Read(buffer)
			value := hex.EncodeToString(buffer)
			generated[field] = value
		} else {
			generated[field] = generateRandomNestedMap(fields, depth-1, bytes)
		}
	}

	return generated
}
