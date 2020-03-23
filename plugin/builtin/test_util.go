package builtin

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

type inputterBenchmark struct {
	fields      int
	depth       int
	fieldLength int
}

func (b inputterBenchmark) String() string {
	return fmt.Sprintf("Fields=%d,Depth=%d,Length=%d", b.fields, b.depth, b.fieldLength)
}

func (b inputterBenchmark) EstimatedBytes() int64 {
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

	return int64(bytes)
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
