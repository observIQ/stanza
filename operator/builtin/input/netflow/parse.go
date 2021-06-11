package netflow

import (
	"net"

	flowmessage "github.com/cloudflare/goflow/v3/pb"
)

func ToMapStringInterface(m *flowmessage.FlowMessage) (map[string]interface{}, error) {
	parsedMap := make(map[string]interface{})

	parsedMap["Type"] = m.GetType()
	parsedMap["TimeReceived"] = m.GetTimeReceived()
	parsedMap["SequenceNum"] = m.GetSequenceNum()
	parsedMap["SamplingRate"] = m.GetSamplingRate()
	parsedMap["FlowDirection"] = m.GetFlowDirection()
	parsedMap["TimeFlowStart"] = m.GetTimeFlowStart()
	parsedMap["TimeFlowEnd"] = m.GetTimeFlowEnd()
	parsedMap["Bytes"] = m.GetBytes()
	parsedMap["Packets"] = m.GetPackets()

	var samplerAddress net.IP = m.GetSamplerAddress()
	parsedMap["SamplerAddress"] = samplerAddress.String()

	var srcAddr net.IP = m.GetSrcAddr()
	parsedMap["SrcAddr"] = srcAddr.String()

	var dstAddr net.IP = m.GetDstAddr()
	parsedMap["DstAddr"] = dstAddr.String()

	var nextHop net.IP = m.GetNextHop()
	parsedMap["NextHop"] = nextHop.String()

	return parsedMap, nil
}

// FlowMessage is a Stanza friendly version of flowprotob FlowMessage
// https://github.com/cloudflare/goflow/blob/master/pb/flow.pb.go
/*type FlowMessage struct {
	Type          flowmessage.FlowMessage_FlowType `json:"Type,omitempty"`
	TimeReceived  uint64                           `json:"TimeReceived,omitempty"`
	SequenceNum   uint32                           `json:"SequenceNum,omitempty"`
	SamplingRate  uint64                           `json:"SamplingRate,omitempty"`
	FlowDirection uint32                           `json:"FlowDirection,omitempty"`
	// Sampler information
	SamplerAddress []byte `json:"SamplerAddress,omitempty"`
	// Found inside packet
	TimeFlowStart uint64 `json:"TimeFlowStart,omitempty"`
	TimeFlowEnd   uint64 `json:"TimeFlowEnd,omitempty"`
	// Size of the sampled packet
	Bytes   uint64 `json:"Bytes,omitempty"`
	Packets uint64 `json:"Packets,omitempty"`
	// Source/destination addresses
	SrcAddr []byte `json:"SrcAddr,omitempty"`
	DstAddr []byte `json:"DstAddr,omitempty"`
	// Layer 3 protocol (IPv4/IPv6/ARP/MPLS...)
	Etype uint32 `json:"Etype,omitempty"`
	// Layer 4 protocol
	Proto uint32 `json:"Proto,omitempty"`
	// Ports for UDP and TCP
	SrcPort uint32 `json:"SrcPort,omitempty"`
	DstPort uint32 `json:"DstPort,omitempty"`
	// Interfaces
	InIf  uint32 `json:"InIf,omitempty"`
	OutIf uint32 `json:"OutIf,omitempty"`
	// Ethernet information
	SrcMac uint64 `json:"SrcMac,omitempty"`
	DstMac uint64 `json:"DstMac,omitempty"`
	// Vlan
	SrcVlan uint32 `json:"SrcVlan,omitempty"`
	DstVlan uint32 `json:"DstVlan,omitempty"`
	// 802.1q VLAN in sampled packet
	VlanId uint32 `json:"VlanId,omitempty"`
	// VRF
	IngressVrfID uint32 `json:"IngressVrfID,omitempty"`
	EgressVrfID  uint32 `json:"EgressVrfID,omitempty"`
	// IP and TCP special flags
	IPTos            uint32 `json:"IPTos,omitempty"`
	ForwardingStatus uint32 `json:"ForwardingStatus,omitempty"`
	IPTTL            uint32 `json:"IPTTL,omitempty"`
	TCPFlags         uint32 `json:"TCPFlags,omitempty"`
	IcmpType         uint32 `json:"IcmpType,omitempty"`
	IcmpCode         uint32 `json:"IcmpCode,omitempty"`
	IPv6FlowLabel    uint32 `json:"IPv6FlowLabel,omitempty"`
	// Fragments (IPv4/IPv6)
	FragmentId      uint32 `json:"FragmentId,omitempty"`
	FragmentOffset  uint32 `json:"FragmentOffset,omitempty"`
	BiFlowDirection uint32 `json:"BiFlowDirection,omitempty"`
	// Autonomous system information
	SrcAS     uint32 `json:"SrcAS,omitempty"`
	DstAS     uint32 `json:"DstAS,omitempty"`
	NextHop   []byte `json:"NextHop,omitempty"`
	NextHopAS uint32 `json:"NextHopAS,omitempty"`
	// Prefix size
	SrcNet uint32 `json:"SrcNet,omitempty"`
	DstNet uint32 `json:"DstNet,omitempty"`
	// IP encapsulation information
	HasEncap            bool   `json:"HasEncap,omitempty"`
	SrcAddrEncap        []byte `json:"SrcAddrEncap,omitempty"`
	DstAddrEncap        []byte `json:"DstAddrEncap,omitempty"`
	ProtoEncap          uint32 `json:"ProtoEncap,omitempty"`
	EtypeEncap          uint32 `json:"EtypeEncap,omitempty"`
	IPTosEncap          uint32 `json:"IPTosEncap,omitempty"`
	IPTTLEncap          uint32 `json:"IPTTLEncap,omitempty"`
	IPv6FlowLabelEncap  uint32 `json:"IPv6FlowLabelEncap,omitempty"`
	FragmentIdEncap     uint32 `json:"FragmentIdEncap,omitempty"`
	FragmentOffsetEncap uint32 `json:"FragmentOffsetEncap,omitempty"`
	// MPLS information
	HasMPLS       bool   `json:"HasMPLS,omitempty"`
	MPLSCount     uint32 `json:"MPLSCount,omitempty"`
	MPLS1TTL      uint32 `json:"MPLS1TTL,omitempty"`
	MPLS1Label    uint32 `json:"MPLS1Label,omitempty"`
	MPLS2TTL      uint32 `json:"MPLS2TTL,omitempty"`
	MPLS2Label    uint32 `json:"MPLS2Label,omitempty"`
	MPLS3TTL      uint32 `json:"MPLS3TTL,omitempty"`
	MPLS3Label    uint32 `json:"MPLS3Label,omitempty"`
	MPLSLastTTL   uint32 `json:"MPLSLastTTL,omitempty"`
	MPLSLastLabel uint32 `json:"MPLSLastLabel,omitempty"`
	// PPP information
	HasPPP               bool     `json:"HasPPP,omitempty"`
	PPPAddressControl    uint32   `json:"PPPAddressControl,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}
*/
