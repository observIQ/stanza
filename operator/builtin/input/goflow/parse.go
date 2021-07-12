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

	// If Proto field exists, add mapped value
	if val, ok := m["Proto"]; ok {
		switch val := val.(type) {
		case uint32:
			const field = "proto_name"
			switch val {
			case 0:
				m[field] = "HOPOPT"
			case 1:
				m[field] = "ICMP"
			case 2:
				m[field] = "IGMP"
			case 3:
				m[field] = "GGP"
			case 4:
				m[field] = "IP-in-IP"
			case 5:
				m[field] = "ST"
			case 6:
				m[field] = "TCP"
			case 7:
				m[field] = "CBT"
			case 8:
				m[field] = "EGP"
			case 9:
				m[field] = "IGP"
			case 10:
				m[field] = "BBN-RCC-MON"
			case 11:
				m[field] = "NVP-II"
			case 12:
				m[field] = "PUP"
			case 13:
				m[field] = "ARGUS"
			case 14:
				m[field] = "EMCON"
			case 15:
				m[field] = "XNET"
			case 16:
				m[field] = "CHAOS"
			case 17:
				m[field] = "UDP"
			case 18:
				m[field] = "MUX"
			case 19:
				m[field] = "DCN-MEAS"
			case 20:
				m[field] = "HMP"
			case 21:
				m[ield] = "PRM"
			case 22:
				m[field] = "XNS-IDP"
			case 23:
				m[field] = "TRUNK-1"
			case 24:
				m[field] = "TRUNK-2"
			case 25:
				m[field] = "LEAF-1"
			case 26:
				m[field] = "LEAF-2"
			case 27:
				m[field] = "RDP"
			case 28:
				m[field] = "IRTP"
			case 29:
				m[field] = "ISO-TP4"
			case 30:
				m[field] = "NETBLT"
			case 31:
				m[field] = "MFE-NSP"
			case 32:
				m[field] = "MERIT-INP"
			case 33:
				m[field] = "DCCP"
			case 34:
				m[field] = "3PC"
			case 35:
				m[field] = "IDPR"
			case 36:
				m[field] = "XTP"
			case 37:
				m[field] = "DDP"
			case 38:
				m[field] = "IDPR-CMTP"
			case 39:
				m[field] = "TP++"
			case 40:
				m[field] = "IL"
			case 41:
				m[field] = "IPv6"
			case 42:
				m[field] = "SDRP"
			case 43:
				m[field] = "IPv6-Route"
			case 44:
				m[field] = "IPv6-Frag"
			case 45:
				m[field] = "IDRP"
			case 46:
				m[field] = "RSVP"
			case 47:
				m[field] = "GRE"
			case 48:
				m[field] = "DSR"
			case 49:
				m[field] = "BNA"
			case 50:
				m[field] = "ESP"
			case 51:
				m[field] = "AH"
			case 52:
				m[field] = "I-NLSP"
			case 53:
				m[field] = "SwIPe"
			case 54:
				m[field] = "NARP"
			case 55:
				m[field] = "MOBILE"
			case 56:
				m[field] = "TLSP"
			case 57:
				m[field] = "SKIP"
			case 58:
				m[field] = "IPv6-ICMP"
			case 59:
				m[field] = "IPv6-NoNxt"
			case 60:
				m[field] = "IPv6-Opts"
			case 61:
				m[field] = "Host Internal Protocol"
			case 62:
				m[field] = "CFTP"
			case 63:
				m[field] = "Local Network"
			case 64:
				m[field] = "SAT-EXPAK"
			case 65:
				m[field] = "KRYPTOLAN"
			case 66:
				m[field] = "RVD"
			case 67:
				m[field] = "IPPC"
			case 68:
				m[field] = "Distribution File System"
			case 69:
				m[field] = "SAT-MON"
			case 70:
				m[field] = "VISA"
			case 71:
				m[field] = "IPCU"
			case 72:
				m[field] = "CPNX"
			case 73:
				m[field] = "CPHB"
			case 74:
				m[field] = "WSN"
			case 75:
				m[field] = "PVP"
			case 76:
				m[field] = "BR-SAT-MON"
			case 77:
				m[field] = "SUN-ND"
			case 78:
				m[field] = "WB-MON"
			case 79:
				m[field] = "WB-EXPAK"
			case 80:
				m[field] = "ISO-IP"
			case 81:
				m[field] = "VMTP"
			case 82:
				m[field] = "SECURE-VMTP"
			case 83:
				m[field] = "VINES"
			case 84:
				m[field] = "TTP/IPTM"
			case 85:
				m[field] = "NSFNET-IGP"
			case 86:
				m[field] = "DGP"
			case 87:
				m[field] = "TCF"
			case 88:
				m[field] = "EIGRP"
			case 89:
				m[field] = "OSPF"
			case 90:
				m[field] = "Sprite-RPC"
			case 91:
				m[field] = "LARP"
			case 92:
				m[field] = "MTP"
			case 93:
				m[field] = "AX.25"
			case 94:
				m[field] = "OS"
			case 95:
				m[field] = "MICP"
			case 96:
				m[field] = "SCC-SP"
			case 97:
				m[field] = "ETHERIP"
			case 98:
				m[field] = "ENCAP"
			case 99:
				m[field] = "Private Encryption"
			case 100:
				m[field] = "GMTP"
			case 101:
				m[field] = "IFMP"
			case 102:
				m[field] = "PNNI"
			case 103:
				m[field] = "PIM"
			case 104:
				m[field] = "ARIS"
			case 105:
				m[field] = "SCPS"
			case 106:
				m[field] = "QNX"
			case 107:
				m[field] = "A/N"
			case 108:
				m[field] = "IPComp"
			case 109:
				m[field] = "SNP"
			case 110:
				m[field] = "Compaq-Peer"
			case 111:
				m[field] = "IPX-in-IP"
			case 112:
				m[field] = "VRRP"
			case 113:
				m[field] = "PGM"
			case 114:
				m[field] = "0-hop Protocol"
			case 115:
				m[field] = "L2TP"
			case 116:
				m[field] = "DDX"
			case 117:
				m[field] = "IATP"
			case 118:
				m[field] = "STP"
			case 119:
				m[field] = "SRP"
			case 120:
				m[field] = "UTI"
			case 121:
				m[field] = "SMP"
			case 122:
				m[field] = "SM"
			case 123:
				m[field] = "PTP"
			case 124:
				m[field] = "IS-IS over IPv4"
			case 125:
				m[field] = "FIRE"
			case 126:
				m[field] = "CRTP"
			case 127:
				m[field] = "CRUDP"
			case 128:
				m[field] = "SSCOPMCE"
			case 129:
				m[field] = "IPLT"
			case 130:
				m[field] = "SPS"
			case 131:
				m[field] = "PIPE"
			case 132:
				m[field] = "SCTP"
			case 133:
				m[field] = "FC"
			case 134:
				m[field] = "RSVP-E2E-IGNORE"
			case 135:
				m[field] = "Mobility Header"
			case 136:
				m[field] = "UDPLite"
			case 137:
				m[field] = "MPLS-in-IP"
			case 138:
				m[field] = "manet"
			case 139:
				m[field] = "HIP"
			case 140:
				m[field] = "Shim6"
			case 141:
				m[field] = "WESP"
			case 142:
				m[field] = "ROHC"
			case 143:
				m[field] = "Ethernet"
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
