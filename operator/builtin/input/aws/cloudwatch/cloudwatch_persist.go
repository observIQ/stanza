package cloudwatch

import (
	"bytes"
	"encoding/binary"

	"github.com/observiq/stanza/operator/helper"
)

// Persister ensures data is persisted across shutdowns
type Persister struct {
	DB helper.Persister
}

// Read is a helper function to get persisted data
func (p *Persister) Read(key string) (int64, error) {
	var startTime int64
	buffer := bytes.NewBuffer(p.DB.Get(key))
	err := binary.Read(buffer, binary.BigEndian, &startTime)
	if err != nil && err.Error() != "EOF" {
		return 0, err
	}
	return startTime, nil
}

// Write is a helper function to set persisted data
func (p *Persister) Write(key string, value int64) {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(value))
	p.DB.Set(key, buf)
}
