package cloudwatch

import (
	"bytes"
	"encoding/binary"

	"github.com/observiq/stanza/operator/helper"
)

type Persister struct {
	DB helper.Persister
}

// Helper function to get persisted data
func (p *Persister) Read(key string) (int64, error) {
	var startTime int64
	buffer := bytes.NewBuffer(p.DB.Get(key))
	err := binary.Read(buffer, binary.BigEndian, &startTime)
	if err != nil && err.Error() != "EOF" {
		return 0, err
	}
	return startTime, nil
}

// Helper function to set persisted data
func (p *Persister) Write(key string, value int64) {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(value))
	p.DB.Set(key, buf)
}
