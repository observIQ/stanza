package goflow

import (
	"net"
	"strings"
	"testing"
	"time"

	flowmessage "github.com/cloudflare/goflow/v3/pb"
	"github.com/stretchr/testify/require"
)

var (
	SamplerAddress = net.IPv4(8, 8, 8, 8)
	SrcAddr        = net.IPv4(1, 1, 1, 1)
	DstAddr        = net.IPv4(10, 1, 1, 1)
	NextHop        = net.IPv4(200, 10, 50, 1)
	SrcAddrEncap   = net.IPv4(1, 10, 2, 1)
	DstAddrEncap   = net.IPv4(10, 10, 10, 10)
)

func TestParse(t *testing.T) {
	cases := []struct {
		name       string
		input      flowmessage.FlowMessage
		expect     map[string]interface{}
		expectTime time.Time
	}{
		{
			"minimal",
			flowmessage.FlowMessage{},
			map[string]interface{}{},
			time.Time{},
		},
		{
			"addresses",
			flowmessage.FlowMessage{
				SamplerAddress: SamplerAddress,
				SrcAddr:        SrcAddr,
				DstAddr:        DstAddr,
				NextHop:        NextHop,
				SrcAddrEncap:   SrcAddrEncap,
				DstAddrEncap:   DstAddrEncap,
			},
			map[string]interface{}{
				"sampleraddress": SamplerAddress.String(),
				"srcaddr":        SrcAddr.String(),
				"dstaddr":        DstAddr.String(),
				"nexthop":        NextHop.String(),
				"srcaddrencap":   SrcAddrEncap.String(),
				"dstaddrencap":   DstAddrEncap.String(),
			},
			time.Time{},
		},
		{
			"promote-time",
			flowmessage.FlowMessage{
				TimeReceived:   1623774351,
				SamplerAddress: SamplerAddress,
				SrcAddr:        SrcAddr,
				DstAddr:        DstAddr,
				NextHop:        NextHop,
				SrcAddrEncap:   SrcAddrEncap,
				DstAddrEncap:   DstAddrEncap,
			},
			map[string]interface{}{
				"sampleraddress": SamplerAddress.String(),
				"srcaddr":        SrcAddr.String(),
				"dstaddr":        DstAddr.String(),
				"nexthop":        NextHop.String(),
				"srcaddrencap":   SrcAddrEncap.String(),
				"dstaddrencap":   DstAddrEncap.String(),
			},
			time.Unix(int64(1623774351), 0),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, outputTime, err := Parse(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.expect, output)

			if tc.input.TimeReceived > 0 {
				require.Equal(t, tc.expectTime, outputTime, "expected field timereceived to be promoted to timestamp")
			}

			for k, _ := range output {
				require.Equal(t, strings.ToLower(k), k, "expected all keys to be lowercase")
			}
		})
	}
}

func BenchmarkRandInt(b *testing.B) {
	m := flowmessage.FlowMessage{
		Type:           flowmessage.FlowMessage_NETFLOW_V5,
		TimeReceived:   uint64(time.Nanosecond),
		SequenceNum:    100,
		SamplingRate:   10,
		FlowDirection:  1,
		SamplerAddress: SamplerAddress,
		TimeFlowStart:  uint64(time.Nanosecond),
		TimeFlowEnd:    uint64(time.Nanosecond),
		Bytes:          100,
		Packets:        200,
		SrcAddr:        SrcAddr,
		DstAddr:        DstAddr,
		Etype:          40,
		Proto:          40,
		InIf:           10,
		OutIf:          1,
		NextHop:        NextHop,
		SrcAddrEncap:   SrcAddrEncap,
		DstAddrEncap:   DstAddrEncap,
	}

	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		_, _, err := Parse(m)
		if err != nil {
			b.FailNow()
		}
	}

}
