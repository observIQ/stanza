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
	ipStringError := "expected ip address to be converted to a string"

	cases := []struct {
		name  string
		input flowmessage.FlowMessage
	}{
		{
			"minimal",
			flowmessage.FlowMessage{},
		},
		{
			"Addresses",
			flowmessage.FlowMessage{
				SamplerAddress: SamplerAddress,
				SrcAddr:        SrcAddr,
				DstAddr:        DstAddr,
				NextHop:        NextHop,
				SrcAddrEncap:   SrcAddrEncap,
				DstAddrEncap:   DstAddrEncap,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := Parse(tc.input)
			require.NoError(t, err)

			for k, v := range output {

				// expect all keys to be lower case
				require.Equal(t, strings.ToLower(k), k)

				// check ip addresses is correct type and value if present
				switch k {
				case "sampleraddress":
					switch v := v.(type) {
					case string:
						require.Equal(t, SamplerAddress.String(), v)
					default:
						require.IsType(t, "string", v, ipStringError)
					}
				case "srcaddr":
					switch v := v.(type) {
					case string:
						require.Equal(t, SrcAddr.String(), v)
					default:
						require.IsType(t, "string", v, ipStringError)
					}
				case "dstaddr":
					switch v := v.(type) {
					case string:
						require.Equal(t, DstAddr.String(), v)
					default:
						require.IsType(t, "string", v, ipStringError)
					}
				case "nexthop":
					switch v := v.(type) {
					case string:
						require.Equal(t, NextHop.String(), v)
					default:
						require.IsType(t, "string", v, ipStringError)
					}
				case "srcaddrencap":
					switch v := v.(type) {
					case string:
						require.Equal(t, SrcAddrEncap.String(), v)
					default:
						require.IsType(t, "string", v, ipStringError)
					}
				case "dstaddrencap":
					switch v := v.(type) {
					case string:
						require.Equal(t, DstAddrEncap.String(), v)
					default:
						require.IsType(t, "string", v, ipStringError)
					}
				}
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
		Parse(m)
	}

}
