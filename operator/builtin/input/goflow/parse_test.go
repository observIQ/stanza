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
		expectErr  bool
	}{
		{
			"minimal",
			flowmessage.FlowMessage{},
			map[string]interface{}{},
			time.Time{},
			false,
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
			false,
		},
		{
			"empty-srcaddr",
			flowmessage.FlowMessage{
				SamplerAddress: SamplerAddress,
				SrcAddr:        []byte{},
				DstAddr:        DstAddr,
				NextHop:        NextHop,
				SrcAddrEncap:   SrcAddrEncap,
				DstAddrEncap:   DstAddrEncap,
			},
			map[string]interface{}{
				"sampleraddress": SamplerAddress.String(),
				"dstaddr":        DstAddr.String(),
				"nexthop":        NextHop.String(),
				"srcaddrencap":   SrcAddrEncap.String(),
				"dstaddrencap":   DstAddrEncap.String(),
			},
			time.Time{},
			false,
		},
		{
			"empty-dstaddr",
			flowmessage.FlowMessage{
				SamplerAddress: SamplerAddress,
				SrcAddr:        SrcAddr,
				DstAddr:        []byte{},
				NextHop:        NextHop,
				SrcAddrEncap:   SrcAddrEncap,
				DstAddrEncap:   DstAddrEncap,
			},
			map[string]interface{}{
				"sampleraddress": SamplerAddress.String(),
				"srcaddr":        SrcAddr.String(),
				"nexthop":        NextHop.String(),
				"srcaddrencap":   SrcAddrEncap.String(),
				"dstaddrencap":   DstAddrEncap.String(),
			},
			time.Time{},
			false,
		},
		{
			"empty-nexthop",
			flowmessage.FlowMessage{
				SamplerAddress: SamplerAddress,
				SrcAddr:        SrcAddr,
				DstAddr:        DstAddr,
				NextHop:        []byte{},
				SrcAddrEncap:   SrcAddrEncap,
				DstAddrEncap:   DstAddrEncap,
			},
			map[string]interface{}{
				"sampleraddress": SamplerAddress.String(),
				"srcaddr":        SrcAddr.String(),
				"dstaddr":        DstAddr.String(),
				"srcaddrencap":   SrcAddrEncap.String(),
				"dstaddrencap":   DstAddrEncap.String(),
			},
			time.Time{},
			false,
		},
		{
			"empty-srcaddrencap",
			flowmessage.FlowMessage{
				SamplerAddress: SamplerAddress,
				SrcAddr:        SrcAddr,
				DstAddr:        DstAddr,
				NextHop:        NextHop,
				SrcAddrEncap:   []byte{},
				DstAddrEncap:   DstAddrEncap,
			},
			map[string]interface{}{
				"sampleraddress": SamplerAddress.String(),
				"srcaddr":        SrcAddr.String(),
				"dstaddr":        DstAddr.String(),
				"nexthop":        NextHop.String(),
				"dstaddrencap":   DstAddrEncap.String(),
			},
			time.Time{},
			false,
		},
		{
			"empty-dstaddrencap",
			flowmessage.FlowMessage{
				SamplerAddress: SamplerAddress,
				SrcAddr:        SrcAddr,
				DstAddr:        DstAddr,
				NextHop:        NextHop,
				SrcAddrEncap:   SrcAddrEncap,
				DstAddrEncap:   []byte{},
			},
			map[string]interface{}{
				"sampleraddress": SamplerAddress.String(),
				"srcaddr":        SrcAddr.String(),
				"dstaddr":        DstAddr.String(),
				"nexthop":        NextHop.String(),
				"srcaddrencap":   SrcAddrEncap.String(),
			},
			time.Time{},
			false,
		},
		{
			"malformed-addresses",
			flowmessage.FlowMessage{
				SamplerAddress: SamplerAddress,
				SrcAddr:        []byte("ip:10.1.1.1"),
				DstAddr:        DstAddr,
				NextHop:        NextHop,
				SrcAddrEncap:   SrcAddrEncap,
				DstAddrEncap:   DstAddrEncap,
			},
			map[string]interface{}{
				"sampleraddress": SamplerAddress.String(),
				"srcaddr":        "ip:10.1.1.1",
				"dstaddr":        DstAddr.String(),
				"nexthop":        NextHop.String(),
				"srcaddrencap":   SrcAddrEncap.String(),
				"dstaddrencap":   DstAddrEncap.String(),
			},
			time.Time{},
			true,
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
			false,
		},
		{
			"proto_name",
			flowmessage.FlowMessage{
				TimeReceived:   1623774351,
				SamplerAddress: SamplerAddress,
				SrcAddr:        SrcAddr,
				DstAddr:        DstAddr,
				NextHop:        NextHop,
				SrcAddrEncap:   SrcAddrEncap,
				DstAddrEncap:   DstAddrEncap,
				Proto:          1,
			},
			map[string]interface{}{
				"sampleraddress": SamplerAddress.String(),
				"srcaddr":        SrcAddr.String(),
				"dstaddr":        DstAddr.String(),
				"nexthop":        NextHop.String(),
				"proto":          uint32(1),
				"proto_name":     "ICMP",
				"srcaddrencap":   SrcAddrEncap.String(),
				"dstaddrencap":   DstAddrEncap.String(),
			},
			time.Unix(int64(1623774351), 0),
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, outputTime, err := Parse(tc.input)
			if tc.expectErr {
				require.Error(t, err)
				return
			}
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

func TestParseProtoName(t *testing.T) {
	cases := []struct {
		name       string
		input      flowmessage.FlowMessage
		expect     map[string]interface{}
		expectTime time.Time
		expectErr  bool
	}{
		{
			"icmp",
			flowmessage.FlowMessage{
				Proto: 1,
			},
			map[string]interface{}{
				"proto":      uint32(1),
				"proto_name": "ICMP",
			},
			time.Time{},
			false,
		},
		{
			"igmp",
			flowmessage.FlowMessage{
				Proto: 2,
			},
			map[string]interface{}{
				"proto":      uint32(2),
				"proto_name": "IGMP",
			},
			time.Time{},
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, _, err := Parse(tc.input)
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expect, output)

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
