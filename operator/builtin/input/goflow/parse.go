package goflow

import (
	"context"
	"fmt"
	"net"
	"strings"

	flowmessage "github.com/cloudflare/goflow/v3/pb"
	"github.com/fatih/structs"
	"github.com/observiq/stanza/errors"
	"go.uber.org/zap"
)

// WriteGoFlowMessage writes netflow messages as entries
func (n *GoflowInput) WriteGoFlowMessage(ctx context.Context, messages []*flowmessage.FlowMessage) {
	n.wg.Add(1)

	for _, msg := range messages {
		m, err := Parse(*msg)
		if err != nil {
			n.Errorf("Failed to parse netflow message", zap.Error(err))
			continue
		}

		entry, err := n.NewEntry(m)
		if err != nil {
			n.Errorf(err.Error())
			continue
		}
		n.Write(ctx, entry)
	}

	n.wg.Done()
}

// Parse parses a netflow message into a map
func Parse(message flowmessage.FlowMessage) (map[string]interface{}, error) {
	structParser := structs.New(message)
	structParser.TagName = "json"
	m := structParser.Map()

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
				ip, err := bytesToIP(x)
				if err != nil {
					return nil, errors.Wrap(err, "error converting DstAddr to string")
				}
				m[key] = ip.String()
			default:
				return nil, fmt.Errorf("type %T cannot be parsed as an IP address", val)
			}
		}
	}

	m = toLower(m)

	return m, nil
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
