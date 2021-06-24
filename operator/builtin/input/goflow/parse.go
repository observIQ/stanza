package goflow

import (
	"fmt"
	"net"
	"strings"
	"time"

	flowmessage "github.com/cloudflare/goflow/v3/pb"
	"github.com/fatih/structs"
	"github.com/observiq/stanza/errors"
)

// Parse parses a netflow message into an entry
func Parse(message flowmessage.FlowMessage) (map[string]interface{}, time.Time, error) {
	structParser := structs.New(message)
	structParser.TagName = "json"
	m := structParser.Map()
	timestamp := time.Time{}

	// https://github.com/cloudflare/goflow/blob/ddd88a7faa89bd9a8e75f0ceca17cbb443c14a8f/pb/flow.pb.go#L57
	// IP address keys are []byte encoded
	byteKeys := [...]string{
		"SamplerAddress",
		"SrcAddr",
		"DstAddr",
		"NextHop",
		"SrcAddrEncap",
		"DstAddrEncap",
	}
	for _, key := range byteKeys {
		if val, ok := m[key]; ok {
			delete(m, key)
			switch x := val.(type) {
			case []byte:
				// If the field is not set, skip it
				if len(x) == 0 {
					continue
				}
				ip, err := bytesToIP(x)
				if err != nil {
					return nil, timestamp, errors.Wrap(err, fmt.Sprintf("error converting %s to string", key))
				}
				m[key] = ip.String()
			default:
				return nil, timestamp, fmt.Errorf("type %T cannot be parsed as an IP address", val)
			}
		}
	}

	m = toLower(m)

	const timeField = "timereceived"
	if val, ok := m[timeField]; ok {
		switch val := val.(type) {
		case uint64:
			timestamp = time.Unix(int64(val), 0)
			delete(m, timeField)
		default:
			return nil, timestamp, fmt.Errorf("failed to promote timestamp, expected %T field %s to be type uint64", timeField, val)
		}
	}

	return m, timestamp, nil
}

// converts all map keys to lowercase
func toLower(m map[string]interface{}) map[string]interface{} {
	x := make(map[string]interface{})
	for k, v := range m {
		x[strings.ToLower(k)] = v
	}
	return x
}

// converts []byte to ip address
func bytesToIP(b []byte) (net.IP, error) {
	switch x := len(b); x {
	case 4, 16:
		var ip net.IP = b
		return ip, nil
	default:
		return nil, fmt.Errorf("cannot convert byte slice to ip address, expected length of 4 or 16 got %d", x)
	}
}
