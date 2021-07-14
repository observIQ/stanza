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
		//{ // known bug, 0 values are being omitted
		//	"hopopt",
		//	flowmessage.FlowMessage{
		//		Proto: 0,
		//	},
		//	map[string]interface{}{
		//		"proto":      uint32(0),
		//		"proto_name": "HOPOPT",
		//	},
		//	time.Time{},
		//	false,
		//},
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
		{
			"ggp",
			flowmessage.FlowMessage{
				Proto: 3,
			},
			map[string]interface{}{
				"proto":      uint32(3),
				"proto_name": "GGP",
			},
			time.Time{},
			false,
		},
		{
			"ip-in-ip",
			flowmessage.FlowMessage{
				Proto: 4,
			},
			map[string]interface{}{
				"proto":      uint32(4),
				"proto_name": "IP-in-IP",
			},
			time.Time{},
			false,
		},
		{
			"st",
			flowmessage.FlowMessage{
				Proto: 5,
			},
			map[string]interface{}{
				"proto":      uint32(5),
				"proto_name": "ST",
			},
			time.Time{},
			false,
		},
		{
			"tcp",
			flowmessage.FlowMessage{
				Proto: 6,
			},
			map[string]interface{}{
				"proto":      uint32(6),
				"proto_name": "TCP",
			},
			time.Time{},
			false,
		},
		{
			"cbt",
			flowmessage.FlowMessage{
				Proto: 7,
			},
			map[string]interface{}{
				"proto":      uint32(7),
				"proto_name": "CBT",
			},
			time.Time{},
			false,
		},
		{
			"egp",
			flowmessage.FlowMessage{
				Proto: 8,
			},
			map[string]interface{}{
				"proto":      uint32(8),
				"proto_name": "EGP",
			},
			time.Time{},
			false,
		},
		{
			"igp",
			flowmessage.FlowMessage{
				Proto: 9,
			},
			map[string]interface{}{
				"proto":      uint32(9),
				"proto_name": "IGP",
			},
			time.Time{},
			false,
		},
		{
			"bbn-rcc-mon",
			flowmessage.FlowMessage{
				Proto: 10,
			},
			map[string]interface{}{
				"proto":      uint32(10),
				"proto_name": "BBN-RCC-MON",
			},
			time.Time{},
			false,
		},
		{
			"nvp-ii",
			flowmessage.FlowMessage{
				Proto: 11,
			},
			map[string]interface{}{
				"proto":      uint32(11),
				"proto_name": "NVP-II",
			},
			time.Time{},
			false,
		},
		{
			"pup",
			flowmessage.FlowMessage{
				Proto: 12,
			},
			map[string]interface{}{
				"proto":      uint32(12),
				"proto_name": "PUP",
			},
			time.Time{},
			false,
		},
		{
			"argus",
			flowmessage.FlowMessage{
				Proto: 13,
			},
			map[string]interface{}{
				"proto":      uint32(13),
				"proto_name": "ARGUS",
			},
			time.Time{},
			false,
		},
		{
			"emcon",
			flowmessage.FlowMessage{
				Proto: 14,
			},
			map[string]interface{}{
				"proto":      uint32(14),
				"proto_name": "EMCON",
			},
			time.Time{},
			false,
		},
		{
			"xnet",
			flowmessage.FlowMessage{
				Proto: 15,
			},
			map[string]interface{}{
				"proto":      uint32(15),
				"proto_name": "XNET",
			},
			time.Time{},
			false,
		},
		{
			"chaos",
			flowmessage.FlowMessage{
				Proto: 16,
			},
			map[string]interface{}{
				"proto":      uint32(16),
				"proto_name": "CHAOS",
			},
			time.Time{},
			false,
		},
		{
			"udp",
			flowmessage.FlowMessage{
				Proto: 17,
			},
			map[string]interface{}{
				"proto":      uint32(17),
				"proto_name": "UDP",
			},
			time.Time{},
			false,
		},
		{
			"mux",
			flowmessage.FlowMessage{
				Proto: 18,
			},
			map[string]interface{}{
				"proto":      uint32(18),
				"proto_name": "MUX",
			},
			time.Time{},
			false,
		},
		{
			"dcn-meas",
			flowmessage.FlowMessage{
				Proto: 19,
			},
			map[string]interface{}{
				"proto":      uint32(19),
				"proto_name": "DCN-MEAS",
			},
			time.Time{},
			false,
		},
		{
			"hmp",
			flowmessage.FlowMessage{
				Proto: 20,
			},
			map[string]interface{}{
				"proto":      uint32(20),
				"proto_name": "HMP",
			},
			time.Time{},
			false,
		},
		{
			"prm",
			flowmessage.FlowMessage{
				Proto: 21,
			},
			map[string]interface{}{
				"proto":      uint32(21),
				"proto_name": "PRM",
			},
			time.Time{},
			false,
		},
		{
			"xns-idp",
			flowmessage.FlowMessage{
				Proto: 22,
			},
			map[string]interface{}{
				"proto":      uint32(22),
				"proto_name": "XNS-IDP",
			},
			time.Time{},
			false,
		},
		{
			"trunk-1",
			flowmessage.FlowMessage{
				Proto: 23,
			},
			map[string]interface{}{
				"proto":      uint32(23),
				"proto_name": "TRUNK-1",
			},
			time.Time{},
			false,
		},
		{
			"trunk-2",
			flowmessage.FlowMessage{
				Proto: 24,
			},
			map[string]interface{}{
				"proto":      uint32(24),
				"proto_name": "TRUNK-2",
			},
			time.Time{},
			false,
		},
		{
			"leaf-1",
			flowmessage.FlowMessage{
				Proto: 25,
			},
			map[string]interface{}{
				"proto":      uint32(25),
				"proto_name": "LEAF-1",
			},
			time.Time{},
			false,
		},
		{
			"leaf-2",
			flowmessage.FlowMessage{
				Proto: 26,
			},
			map[string]interface{}{
				"proto":      uint32(26),
				"proto_name": "LEAF-2",
			},
			time.Time{},
			false,
		},
		{
			"rdp",
			flowmessage.FlowMessage{
				Proto: 27,
			},
			map[string]interface{}{
				"proto":      uint32(27),
				"proto_name": "RDP",
			},
			time.Time{},
			false,
		},
		{
			"irtp",
			flowmessage.FlowMessage{
				Proto: 28,
			},
			map[string]interface{}{
				"proto":      uint32(28),
				"proto_name": "IRTP",
			},
			time.Time{},
			false,
		},
		{
			"iso-tp4",
			flowmessage.FlowMessage{
				Proto: 29,
			},
			map[string]interface{}{
				"proto":      uint32(29),
				"proto_name": "ISO-TP4",
			},
			time.Time{},
			false,
		},
		{
			"netblt",
			flowmessage.FlowMessage{
				Proto: 30,
			},
			map[string]interface{}{
				"proto":      uint32(30),
				"proto_name": "NETBLT",
			},
			time.Time{},
			false,
		},
		{
			"mfe-nsp",
			flowmessage.FlowMessage{
				Proto: 31,
			},
			map[string]interface{}{
				"proto":      uint32(31),
				"proto_name": "MFE-NSP",
			},
			time.Time{},
			false,
		},
		{
			"merit-inp",
			flowmessage.FlowMessage{
				Proto: 32,
			},
			map[string]interface{}{
				"proto":      uint32(32),
				"proto_name": "MERIT-INP",
			},
			time.Time{},
			false,
		},
		{
			"dccp",
			flowmessage.FlowMessage{
				Proto: 33,
			},
			map[string]interface{}{
				"proto":      uint32(33),
				"proto_name": "DCCP",
			},
			time.Time{},
			false,
		},
		{
			"3pc",
			flowmessage.FlowMessage{
				Proto: 34,
			},
			map[string]interface{}{
				"proto":      uint32(34),
				"proto_name": "3PC",
			},
			time.Time{},
			false,
		},
		{
			"idpr",
			flowmessage.FlowMessage{
				Proto: 35,
			},
			map[string]interface{}{
				"proto":      uint32(35),
				"proto_name": "IDPR",
			},
			time.Time{},
			false,
		},
		{
			"xtp",
			flowmessage.FlowMessage{
				Proto: 36,
			},
			map[string]interface{}{
				"proto":      uint32(36),
				"proto_name": "XTP",
			},
			time.Time{},
			false,
		},
		{
			"ddp",
			flowmessage.FlowMessage{
				Proto: 37,
			},
			map[string]interface{}{
				"proto":      uint32(37),
				"proto_name": "DDP",
			},
			time.Time{},
			false,
		},
		{
			"idpr-cmtp",
			flowmessage.FlowMessage{
				Proto: 38,
			},
			map[string]interface{}{
				"proto":      uint32(38),
				"proto_name": "IDPR-CMTP",
			},
			time.Time{},
			false,
		},
		{
			"tp++",
			flowmessage.FlowMessage{
				Proto: 39,
			},
			map[string]interface{}{
				"proto":      uint32(39),
				"proto_name": "TP++",
			},
			time.Time{},
			false,
		},
		{
			"il",
			flowmessage.FlowMessage{
				Proto: 40,
			},
			map[string]interface{}{
				"proto":      uint32(40),
				"proto_name": "IL",
			},
			time.Time{},
			false,
		},
		{
			"ipv6",
			flowmessage.FlowMessage{
				Proto: 41,
			},
			map[string]interface{}{
				"proto":      uint32(41),
				"proto_name": "IPv6",
			},
			time.Time{},
			false,
		},
		{
			"sdrp",
			flowmessage.FlowMessage{
				Proto: 42,
			},
			map[string]interface{}{
				"proto":      uint32(42),
				"proto_name": "SDRP",
			},
			time.Time{},
			false,
		},
		{
			"ipv6-route",
			flowmessage.FlowMessage{
				Proto: 43,
			},
			map[string]interface{}{
				"proto":      uint32(43),
				"proto_name": "IPv6-Route",
			},
			time.Time{},
			false,
		},
		{
			"ipv6-frag",
			flowmessage.FlowMessage{
				Proto: 44,
			},
			map[string]interface{}{
				"proto":      uint32(44),
				"proto_name": "IPv6-Frag",
			},
			time.Time{},
			false,
		},
		{
			"idrp",
			flowmessage.FlowMessage{
				Proto: 45,
			},
			map[string]interface{}{
				"proto":      uint32(45),
				"proto_name": "IDRP",
			},
			time.Time{},
			false,
		},
		{
			"rsvp",
			flowmessage.FlowMessage{
				Proto: 46,
			},
			map[string]interface{}{
				"proto":      uint32(46),
				"proto_name": "RSVP",
			},
			time.Time{},
			false,
		},
		{
			"gre",
			flowmessage.FlowMessage{
				Proto: 47,
			},
			map[string]interface{}{
				"proto":      uint32(47),
				"proto_name": "GRE",
			},
			time.Time{},
			false,
		},
		{
			"dsr",
			flowmessage.FlowMessage{
				Proto: 48,
			},
			map[string]interface{}{
				"proto":      uint32(48),
				"proto_name": "DSR",
			},
			time.Time{},
			false,
		},
		{
			"bna",
			flowmessage.FlowMessage{
				Proto: 49,
			},
			map[string]interface{}{
				"proto":      uint32(49),
				"proto_name": "BNA",
			},
			time.Time{},
			false,
		},
		{
			"esp",
			flowmessage.FlowMessage{
				Proto: 50,
			},
			map[string]interface{}{
				"proto":      uint32(50),
				"proto_name": "ESP",
			},
			time.Time{},
			false,
		},
		{
			"ah",
			flowmessage.FlowMessage{
				Proto: 51,
			},
			map[string]interface{}{
				"proto":      uint32(51),
				"proto_name": "AH",
			},
			time.Time{},
			false,
		},
		{
			"i-nlsp",
			flowmessage.FlowMessage{
				Proto: 52,
			},
			map[string]interface{}{
				"proto":      uint32(52),
				"proto_name": "I-NLSP",
			},
			time.Time{},
			false,
		},
		{
			"swipe",
			flowmessage.FlowMessage{
				Proto: 53,
			},
			map[string]interface{}{
				"proto":      uint32(53),
				"proto_name": "SwIPe",
			},
			time.Time{},
			false,
		},
		{
			"narp",
			flowmessage.FlowMessage{
				Proto: 54,
			},
			map[string]interface{}{
				"proto":      uint32(54),
				"proto_name": "NARP",
			},
			time.Time{},
			false,
		},
		{
			"mobile",
			flowmessage.FlowMessage{
				Proto: 55,
			},
			map[string]interface{}{
				"proto":      uint32(55),
				"proto_name": "MOBILE",
			},
			time.Time{},
			false,
		},
		{
			"tlsp",
			flowmessage.FlowMessage{
				Proto: 56,
			},
			map[string]interface{}{
				"proto":      uint32(56),
				"proto_name": "TLSP",
			},
			time.Time{},
			false,
		},
		{
			"skip",
			flowmessage.FlowMessage{
				Proto: 57,
			},
			map[string]interface{}{
				"proto":      uint32(57),
				"proto_name": "SKIP",
			},
			time.Time{},
			false,
		},
		{
			"ipv6-icmp",
			flowmessage.FlowMessage{
				Proto: 58,
			},
			map[string]interface{}{
				"proto":      uint32(58),
				"proto_name": "IPv6-ICMP",
			},
			time.Time{},
			false,
		},
		{
			"ipv6-nonxt",
			flowmessage.FlowMessage{
				Proto: 59,
			},
			map[string]interface{}{
				"proto":      uint32(59),
				"proto_name": "IPv6-NoNxt",
			},
			time.Time{},
			false,
		},
		{
			"ipv6-opts",
			flowmessage.FlowMessage{
				Proto: 60,
			},
			map[string]interface{}{
				"proto":      uint32(60),
				"proto_name": "IPv6-Opts",
			},
			time.Time{},
			false,
		},
		{
			"host internal protocol",
			flowmessage.FlowMessage{
				Proto: 61,
			},
			map[string]interface{}{
				"proto":      uint32(61),
				"proto_name": "Host Internal Protocol",
			},
			time.Time{},
			false,
		},
		{
			"cftp",
			flowmessage.FlowMessage{
				Proto: 62,
			},
			map[string]interface{}{
				"proto":      uint32(62),
				"proto_name": "CFTP",
			},
			time.Time{},
			false,
		},
		{
			"local network",
			flowmessage.FlowMessage{
				Proto: 63,
			},
			map[string]interface{}{
				"proto":      uint32(63),
				"proto_name": "Local Network",
			},
			time.Time{},
			false,
		},
		{
			"sat-expak",
			flowmessage.FlowMessage{
				Proto: 64,
			},
			map[string]interface{}{
				"proto":      uint32(64),
				"proto_name": "SAT-EXPAK",
			},
			time.Time{},
			false,
		},
		{
			"kryptolan",
			flowmessage.FlowMessage{
				Proto: 65,
			},
			map[string]interface{}{
				"proto":      uint32(65),
				"proto_name": "KRYPTOLAN",
			},
			time.Time{},
			false,
		},
		{
			"rvd",
			flowmessage.FlowMessage{
				Proto: 66,
			},
			map[string]interface{}{
				"proto":      uint32(66),
				"proto_name": "RVD",
			},
			time.Time{},
			false,
		},
		{
			"ippc",
			flowmessage.FlowMessage{
				Proto: 67,
			},
			map[string]interface{}{
				"proto":      uint32(67),
				"proto_name": "IPPC",
			},
			time.Time{},
			false,
		},
		{
			"distribution file system",
			flowmessage.FlowMessage{
				Proto: 68,
			},
			map[string]interface{}{
				"proto":      uint32(68),
				"proto_name": "Distribution File System",
			},
			time.Time{},
			false,
		},
		{
			"sat-mon",
			flowmessage.FlowMessage{
				Proto: 69,
			},
			map[string]interface{}{
				"proto":      uint32(69),
				"proto_name": "SAT-MON",
			},
			time.Time{},
			false,
		},
		{
			"visa",
			flowmessage.FlowMessage{
				Proto: 70,
			},
			map[string]interface{}{
				"proto":      uint32(70),
				"proto_name": "VISA",
			},
			time.Time{},
			false,
		},
		{
			"ipcu",
			flowmessage.FlowMessage{
				Proto: 71,
			},
			map[string]interface{}{
				"proto":      uint32(71),
				"proto_name": "IPCU",
			},
			time.Time{},
			false,
		},
		{
			"cpnx",
			flowmessage.FlowMessage{
				Proto: 72,
			},
			map[string]interface{}{
				"proto":      uint32(72),
				"proto_name": "CPNX",
			},
			time.Time{},
			false,
		},
		{
			"cphb",
			flowmessage.FlowMessage{
				Proto: 73,
			},
			map[string]interface{}{
				"proto":      uint32(73),
				"proto_name": "CPHB",
			},
			time.Time{},
			false,
		},
		{
			"wsn",
			flowmessage.FlowMessage{
				Proto: 74,
			},
			map[string]interface{}{
				"proto":      uint32(74),
				"proto_name": "WSN",
			},
			time.Time{},
			false,
		},
		{
			"pvp",
			flowmessage.FlowMessage{
				Proto: 75,
			},
			map[string]interface{}{
				"proto":      uint32(75),
				"proto_name": "PVP",
			},
			time.Time{},
			false,
		},
		{
			"br-sat-mon",
			flowmessage.FlowMessage{
				Proto: 76,
			},
			map[string]interface{}{
				"proto":      uint32(76),
				"proto_name": "BR-SAT-MON",
			},
			time.Time{},
			false,
		},
		{
			"sun-nd",
			flowmessage.FlowMessage{
				Proto: 77,
			},
			map[string]interface{}{
				"proto":      uint32(77),
				"proto_name": "SUN-ND",
			},
			time.Time{},
			false,
		},
		{
			"wb-mon",
			flowmessage.FlowMessage{
				Proto: 78,
			},
			map[string]interface{}{
				"proto":      uint32(78),
				"proto_name": "WB-MON",
			},
			time.Time{},
			false,
		},
		{
			"wb-expak",
			flowmessage.FlowMessage{
				Proto: 79,
			},
			map[string]interface{}{
				"proto":      uint32(79),
				"proto_name": "WB-EXPAK",
			},
			time.Time{},
			false,
		},
		{
			"iso-ip",
			flowmessage.FlowMessage{
				Proto: 80,
			},
			map[string]interface{}{
				"proto":      uint32(80),
				"proto_name": "ISO-IP",
			},
			time.Time{},
			false,
		},
		{
			"vmtp",
			flowmessage.FlowMessage{
				Proto: 81,
			},
			map[string]interface{}{
				"proto":      uint32(81),
				"proto_name": "VMTP",
			},
			time.Time{},
			false,
		},
		{
			"secure-vmtp",
			flowmessage.FlowMessage{
				Proto: 82,
			},
			map[string]interface{}{
				"proto":      uint32(82),
				"proto_name": "SECURE-VMTP",
			},
			time.Time{},
			false,
		},
		{
			"vines",
			flowmessage.FlowMessage{
				Proto: 83,
			},
			map[string]interface{}{
				"proto":      uint32(83),
				"proto_name": "VINES",
			},
			time.Time{},
			false,
		},
		{
			"ttp/iptm",
			flowmessage.FlowMessage{
				Proto: 84,
			},
			map[string]interface{}{
				"proto":      uint32(84),
				"proto_name": "TTP/IPTM",
			},
			time.Time{},
			false,
		},
		{
			"nsfnet-igp",
			flowmessage.FlowMessage{
				Proto: 85,
			},
			map[string]interface{}{
				"proto":      uint32(85),
				"proto_name": "NSFNET-IGP",
			},
			time.Time{},
			false,
		},
		{
			"dgp",
			flowmessage.FlowMessage{
				Proto: 86,
			},
			map[string]interface{}{
				"proto":      uint32(86),
				"proto_name": "DGP",
			},
			time.Time{},
			false,
		},
		{
			"tcf",
			flowmessage.FlowMessage{
				Proto: 87,
			},
			map[string]interface{}{
				"proto":      uint32(87),
				"proto_name": "TCF",
			},
			time.Time{},
			false,
		},
		{
			"eigrp",
			flowmessage.FlowMessage{
				Proto: 88,
			},
			map[string]interface{}{
				"proto":      uint32(88),
				"proto_name": "EIGRP",
			},
			time.Time{},
			false,
		},
		{
			"ospf",
			flowmessage.FlowMessage{
				Proto: 89,
			},
			map[string]interface{}{
				"proto":      uint32(89),
				"proto_name": "OSPF",
			},
			time.Time{},
			false,
		},
		{
			"sprite-rpc",
			flowmessage.FlowMessage{
				Proto: 90,
			},
			map[string]interface{}{
				"proto":      uint32(90),
				"proto_name": "Sprite-RPC",
			},
			time.Time{},
			false,
		},
		{
			"larp",
			flowmessage.FlowMessage{
				Proto: 91,
			},
			map[string]interface{}{
				"proto":      uint32(91),
				"proto_name": "LARP",
			},
			time.Time{},
			false,
		},
		{
			"mtp",
			flowmessage.FlowMessage{
				Proto: 92,
			},
			map[string]interface{}{
				"proto":      uint32(92),
				"proto_name": "MTP",
			},
			time.Time{},
			false,
		},
		{
			"ax.25",
			flowmessage.FlowMessage{
				Proto: 93,
			},
			map[string]interface{}{
				"proto":      uint32(93),
				"proto_name": "AX.25",
			},
			time.Time{},
			false,
		},
		{
			"os",
			flowmessage.FlowMessage{
				Proto: 94,
			},
			map[string]interface{}{
				"proto":      uint32(94),
				"proto_name": "OS",
			},
			time.Time{},
			false,
		},
		{
			"micp",
			flowmessage.FlowMessage{
				Proto: 95,
			},
			map[string]interface{}{
				"proto":      uint32(95),
				"proto_name": "MICP",
			},
			time.Time{},
			false,
		},
		{
			"scc-sp",
			flowmessage.FlowMessage{
				Proto: 96,
			},
			map[string]interface{}{
				"proto":      uint32(96),
				"proto_name": "SCC-SP",
			},
			time.Time{},
			false,
		},
		{
			"etherip",
			flowmessage.FlowMessage{
				Proto: 97,
			},
			map[string]interface{}{
				"proto":      uint32(97),
				"proto_name": "ETHERIP",
			},
			time.Time{},
			false,
		},
		{
			"encap",
			flowmessage.FlowMessage{
				Proto: 98,
			},
			map[string]interface{}{
				"proto":      uint32(98),
				"proto_name": "ENCAP",
			},
			time.Time{},
			false,
		},
		{
			"private encryption",
			flowmessage.FlowMessage{
				Proto: 99,
			},
			map[string]interface{}{
				"proto":      uint32(99),
				"proto_name": "Private Encryption",
			},
			time.Time{},
			false,
		},
		{
			"gmtp",
			flowmessage.FlowMessage{
				Proto: 100,
			},
			map[string]interface{}{
				"proto":      uint32(100),
				"proto_name": "GMTP",
			},
			time.Time{},
			false,
		},
		{
			"ifmp",
			flowmessage.FlowMessage{
				Proto: 101,
			},
			map[string]interface{}{
				"proto":      uint32(101),
				"proto_name": "IFMP",
			},
			time.Time{},
			false,
		},
		{
			"pnni",
			flowmessage.FlowMessage{
				Proto: 102,
			},
			map[string]interface{}{
				"proto":      uint32(102),
				"proto_name": "PNNI",
			},
			time.Time{},
			false,
		},
		{
			"pim",
			flowmessage.FlowMessage{
				Proto: 103,
			},
			map[string]interface{}{
				"proto":      uint32(103),
				"proto_name": "PIM",
			},
			time.Time{},
			false,
		},
		{
			"aris",
			flowmessage.FlowMessage{
				Proto: 104,
			},
			map[string]interface{}{
				"proto":      uint32(104),
				"proto_name": "ARIS",
			},
			time.Time{},
			false,
		},
		{
			"scps",
			flowmessage.FlowMessage{
				Proto: 105,
			},
			map[string]interface{}{
				"proto":      uint32(105),
				"proto_name": "SCPS",
			},
			time.Time{},
			false,
		},
		{
			"qnx",
			flowmessage.FlowMessage{
				Proto: 106,
			},
			map[string]interface{}{
				"proto":      uint32(106),
				"proto_name": "QNX",
			},
			time.Time{},
			false,
		},
		{
			"a/n",
			flowmessage.FlowMessage{
				Proto: 107,
			},
			map[string]interface{}{
				"proto":      uint32(107),
				"proto_name": "A/N",
			},
			time.Time{},
			false,
		},
		{
			"ipcomp",
			flowmessage.FlowMessage{
				Proto: 108,
			},
			map[string]interface{}{
				"proto":      uint32(108),
				"proto_name": "IPComp",
			},
			time.Time{},
			false,
		},
		{
			"snp",
			flowmessage.FlowMessage{
				Proto: 109,
			},
			map[string]interface{}{
				"proto":      uint32(109),
				"proto_name": "SNP",
			},
			time.Time{},
			false,
		},
		{
			"compaq-peer",
			flowmessage.FlowMessage{
				Proto: 110,
			},
			map[string]interface{}{
				"proto":      uint32(110),
				"proto_name": "Compaq-Peer",
			},
			time.Time{},
			false,
		},
		{
			"ipx-in-ip",
			flowmessage.FlowMessage{
				Proto: 111,
			},
			map[string]interface{}{
				"proto":      uint32(111),
				"proto_name": "IPX-in-IP",
			},
			time.Time{},
			false,
		},
		{
			"vrrp",
			flowmessage.FlowMessage{
				Proto: 112,
			},
			map[string]interface{}{
				"proto":      uint32(112),
				"proto_name": "VRRP",
			},
			time.Time{},
			false,
		},
		{
			"pgm",
			flowmessage.FlowMessage{
				Proto: 113,
			},
			map[string]interface{}{
				"proto":      uint32(113),
				"proto_name": "PGM",
			},
			time.Time{},
			false,
		},
		{
			"0-hop protocol",
			flowmessage.FlowMessage{
				Proto: 114,
			},
			map[string]interface{}{
				"proto":      uint32(114),
				"proto_name": "0-hop Protocol",
			},
			time.Time{},
			false,
		},
		{
			"l2tp",
			flowmessage.FlowMessage{
				Proto: 115,
			},
			map[string]interface{}{
				"proto":      uint32(115),
				"proto_name": "L2TP",
			},
			time.Time{},
			false,
		},
		{
			"ddx",
			flowmessage.FlowMessage{
				Proto: 116,
			},
			map[string]interface{}{
				"proto":      uint32(116),
				"proto_name": "DDX",
			},
			time.Time{},
			false,
		},
		{
			"iatp",
			flowmessage.FlowMessage{
				Proto: 117,
			},
			map[string]interface{}{
				"proto":      uint32(117),
				"proto_name": "IATP",
			},
			time.Time{},
			false,
		},
		{
			"stp",
			flowmessage.FlowMessage{
				Proto: 118,
			},
			map[string]interface{}{
				"proto":      uint32(118),
				"proto_name": "STP",
			},
			time.Time{},
			false,
		},
		{
			"srp",
			flowmessage.FlowMessage{
				Proto: 119,
			},
			map[string]interface{}{
				"proto":      uint32(119),
				"proto_name": "SRP",
			},
			time.Time{},
			false,
		},
		{
			"uti",
			flowmessage.FlowMessage{
				Proto: 120,
			},
			map[string]interface{}{
				"proto":      uint32(120),
				"proto_name": "UTI",
			},
			time.Time{},
			false,
		},
		{
			"smp",
			flowmessage.FlowMessage{
				Proto: 121,
			},
			map[string]interface{}{
				"proto":      uint32(121),
				"proto_name": "SMP",
			},
			time.Time{},
			false,
		},
		{
			"sm",
			flowmessage.FlowMessage{
				Proto: 122,
			},
			map[string]interface{}{
				"proto":      uint32(122),
				"proto_name": "SM",
			},
			time.Time{},
			false,
		},
		{
			"ptp",
			flowmessage.FlowMessage{
				Proto: 123,
			},
			map[string]interface{}{
				"proto":      uint32(123),
				"proto_name": "PTP",
			},
			time.Time{},
			false,
		},
		{
			"is-is over ipv4",
			flowmessage.FlowMessage{
				Proto: 124,
			},
			map[string]interface{}{
				"proto":      uint32(124),
				"proto_name": "IS-IS over IPv4",
			},
			time.Time{},
			false,
		},
		{
			"fire",
			flowmessage.FlowMessage{
				Proto: 125,
			},
			map[string]interface{}{
				"proto":      uint32(125),
				"proto_name": "FIRE",
			},
			time.Time{},
			false,
		},
		{
			"crtp",
			flowmessage.FlowMessage{
				Proto: 126,
			},
			map[string]interface{}{
				"proto":      uint32(126),
				"proto_name": "CRTP",
			},
			time.Time{},
			false,
		},
		{
			"crudp",
			flowmessage.FlowMessage{
				Proto: 127,
			},
			map[string]interface{}{
				"proto":      uint32(127),
				"proto_name": "CRUDP",
			},
			time.Time{},
			false,
		},
		{
			"sscopmce",
			flowmessage.FlowMessage{
				Proto: 128,
			},
			map[string]interface{}{
				"proto":      uint32(128),
				"proto_name": "SSCOPMCE",
			},
			time.Time{},
			false,
		},
		{
			"iplt",
			flowmessage.FlowMessage{
				Proto: 129,
			},
			map[string]interface{}{
				"proto":      uint32(129),
				"proto_name": "IPLT",
			},
			time.Time{},
			false,
		},
		{
			"sps",
			flowmessage.FlowMessage{
				Proto: 130,
			},
			map[string]interface{}{
				"proto":      uint32(130),
				"proto_name": "SPS",
			},
			time.Time{},
			false,
		},
		{
			"pipe",
			flowmessage.FlowMessage{
				Proto: 131,
			},
			map[string]interface{}{
				"proto":      uint32(131),
				"proto_name": "PIPE",
			},
			time.Time{},
			false,
		},
		{
			"sctp",
			flowmessage.FlowMessage{
				Proto: 132,
			},
			map[string]interface{}{
				"proto":      uint32(132),
				"proto_name": "SCTP",
			},
			time.Time{},
			false,
		},
		{
			"fc",
			flowmessage.FlowMessage{
				Proto: 133,
			},
			map[string]interface{}{
				"proto":      uint32(133),
				"proto_name": "FC",
			},
			time.Time{},
			false,
		},
		{
			"rsvp-e2e-ignore",
			flowmessage.FlowMessage{
				Proto: 134,
			},
			map[string]interface{}{
				"proto":      uint32(134),
				"proto_name": "RSVP-E2E-IGNORE",
			},
			time.Time{},
			false,
		},
		{
			"mobility header",
			flowmessage.FlowMessage{
				Proto: 135,
			},
			map[string]interface{}{
				"proto":      uint32(135),
				"proto_name": "Mobility Header",
			},
			time.Time{},
			false,
		},
		{
			"udplite",
			flowmessage.FlowMessage{
				Proto: 136,
			},
			map[string]interface{}{
				"proto":      uint32(136),
				"proto_name": "UDPLite",
			},
			time.Time{},
			false,
		},
		{
			"mpls-in-ip",
			flowmessage.FlowMessage{
				Proto: 137,
			},
			map[string]interface{}{
				"proto":      uint32(137),
				"proto_name": "MPLS-in-IP",
			},
			time.Time{},
			false,
		},
		{
			"manet",
			flowmessage.FlowMessage{
				Proto: 138,
			},
			map[string]interface{}{
				"proto":      uint32(138),
				"proto_name": "manet",
			},
			time.Time{},
			false,
		},
		{
			"hip",
			flowmessage.FlowMessage{
				Proto: 139,
			},
			map[string]interface{}{
				"proto":      uint32(139),
				"proto_name": "HIP",
			},
			time.Time{},
			false,
		},
		{
			"shim6",
			flowmessage.FlowMessage{
				Proto: 140,
			},
			map[string]interface{}{
				"proto":      uint32(140),
				"proto_name": "Shim6",
			},
			time.Time{},
			false,
		},
		{
			"wesp",
			flowmessage.FlowMessage{
				Proto: 141,
			},
			map[string]interface{}{
				"proto":      uint32(141),
				"proto_name": "WESP",
			},
			time.Time{},
			false,
		},
		{
			"rohc",
			flowmessage.FlowMessage{
				Proto: 142,
			},
			map[string]interface{}{
				"proto":      uint32(142),
				"proto_name": "ROHC",
			},
			time.Time{},
			false,
		},
		{
			"ethernet",
			flowmessage.FlowMessage{
				Proto: 143,
			},
			map[string]interface{}{
				"proto":      uint32(143),
				"proto_name": "Ethernet",
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

func BenchmarkParse(b *testing.B) {
	m := flowmessage.FlowMessage{
		Type:                flowmessage.FlowMessage_NETFLOW_V5,
		TimeReceived:        uint64(time.Nanosecond),
		SequenceNum:         100,
		SamplingRate:        10,
		FlowDirection:       1,
		SamplerAddress:      SamplerAddress,
		TimeFlowStart:       uint64(time.Nanosecond),
		TimeFlowEnd:         uint64(time.Nanosecond),
		Bytes:               100,
		Packets:             200,
		SrcAddr:             SrcAddr,
		DstAddr:             DstAddr,
		Etype:               40,
		Proto:               40,
		SrcPort:             2,
		DstPort:             40,
		InIf:                10,
		OutIf:               1,
		SrcMac:              100000000,
		DstMac:              100000000,
		SrcVlan:             10,
		DstVlan:             20,
		VlanId:              4,
		IngressVrfID:        4,
		EgressVrfID:         0,
		IPTos:               1,
		ForwardingStatus:    4,
		IPTTL:               5,
		TCPFlags:            5,
		IcmpType:            5,
		IcmpCode:            5,
		IPv6FlowLabel:       5,
		FragmentId:          5,
		FragmentOffset:      5,
		BiFlowDirection:     5,
		SrcAS:               5,
		DstAS:               5,
		NextHop:             SamplerAddress,
		NextHopAS:           5,
		SrcNet:              24,
		DstNet:              16,
		HasEncap:            true,
		SrcAddrEncap:        SrcAddrEncap,
		DstAddrEncap:        DstAddrEncap,
		ProtoEncap:          5,
		EtypeEncap:          5,
		IPTosEncap:          5,
		IPTTLEncap:          5,
		IPv6FlowLabelEncap:  5,
		FragmentIdEncap:     5,
		FragmentOffsetEncap: 5,
		HasMPLS:             true,
		MPLSCount:           5,
		MPLS1TTL:            5,
		MPLS1Label:          5,
		MPLS2TTL:            5,
		MPLS2Label:          5,
		MPLS3TTL:            5,
		MPLS3Label:          5,
		MPLSLastTTL:         5,
		MPLSLastLabel:       5,
		HasPPP:              false,
		PPPAddressControl:   0,
	}

	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		_, _, err := Parse(m)
		if err != nil {
			b.Errorf(err.Error())
			b.FailNow()
		}
	}

}
