package main

import (
	"bytes"
	"fmt"
	"net"
	"unsafe"
)

const IPV6_MULTICAST_ADDRESS = "ff02::1:ff00:0000"

const ICMPV6_TYPE_ECHO_REQUEST uint8 = 128
const ICMPV6_TYPE_ECHO_REPLY uint8 = 129
const ICMPV6_TYPE_ROUTER_SOLICIATION uint8 = 133
const ICMPV6_TYPE_NEIGHBOR_SOLICIATION uint8 = 135
const ICMPV6_TYPE_NEIGHBOR_ADVERTISEMENT uint8 = 136

const ICMPV6_NA_FLAG_SOLICITED uint8 = 0b01000000
const ICMPV6_NA_FLAG_OVERRIDE uint8 = 0b00100000

type icmpv6Hdr struct {
	icmpType uint8
	code     uint8
	checksum uint16
}

type icmpv6Echo struct {
	header icmpv6Hdr
	id     uint16
	seq    uint16
	data   []uint8
}

/**
 * Neighbor Advertisement（NA）
 * http://www.tcpipguide.com/free/t_ICMPv6NeighborAdvertisementandNeighborSolicitation-2.htm
 */
type icmpv6Na struct {
	hdr        icmpv6Hdr
	flags      uint32
	targetAddr in6Addr
	optType    uint8
	optLength  uint8
	optMacAddr [6]uint8
}

/* ICMPv6パケットの受信処理 */
func icmpv6Input(netDev *netDevice, srcAddr in6Addr, dstAddr in6Addr, icmpPacket []byte) {
	if len(icmpPacket) < 4 {
		fmt.Printf("received ICMP Packet is too short. size is %d", len(icmpPacket))
		return
	}
	recIcmpHdr := icmpv6Hdr{
		icmpType: icmpPacket[0],
		code:     icmpPacket[1],
		checksum: byteToUint16(icmpPacket[2:4]),
	}
	fmt.Printf("received icmpv6 code=%d, type=%d\n", recIcmpHdr.code, recIcmpHdr.icmpType)

	switch recIcmpHdr.icmpType {
	case ICMPV6_TYPE_NEIGHBOR_SOLICIATION:
		if len(icmpPacket) < 32 {
			fmt.Printf("received icmpv6 NS Packet is too short. size is %d", len(icmpPacket))
			return
		}
		targetAddr := in6Addr(icmpPacket[8:24])
		targetAddrStr := fmtIpStr(targetAddr)
		fmt.Printf("icmpv6 NS packet. targetAddr is %s\n", targetAddrStr)

		nsPkt := icmpv6Na{
			hdr:        recIcmpHdr,
			flags:      byteToUint32(icmpPacket[4:8]),
			targetAddr: targetAddr,
			optType:    icmpPacket[24],
			optLength:  icmpPacket[25],
			optMacAddr: [6]uint8(icmpPacket[26:32]),
		}

		if netDev.ipv6Dev.address == nsPkt.targetAddr {
			fmt.Printf("ns target match! %s\n", targetAddrStr)
			fmt.Printf("option mac address! %s\n", fmtMacStr(nsPkt.optMacAddr))

			updateNDTableEntry(netDev, nsPkt.optMacAddr, srcAddr)

			naPkt := icmpv6Na{
				hdr: icmpv6Hdr{
					icmpType: ICMPV6_TYPE_NEIGHBOR_ADVERTISEMENT,
					code:     0,
					checksum: 0,
				},
				flags:      byteToUint32([]byte{ICMPV6_NA_FLAG_SOLICITED | ICMPV6_NA_FLAG_OVERRIDE, 0x00, 0x00, 0x00}),
				targetAddr: nsPkt.targetAddr,
				optType:    2,
				optLength:  1,
				optMacAddr: netDev.macAddr,
			}

			phdr := ipv6PseudoHeader{
				srcAddr:      netDev.ipv6Dev.address,
				dstAddr:      srcAddr,
				packetLength: uint32(unsafe.Sizeof(icmpv6Na{})), // nits: 直書きでいいかも
				zero:         [3]byte{0x00, 0x00, 0x00},
				nextHeader:   IPV6_PROTOCOL_NUM_ICMP,
			}

			psum := checksum16(phdr.toPseudoHeader(), 0)
			naPkt.hdr.checksum = checksum16(naPkt.icmpv6NaToPacket(), psum^0xffff)

			ipv6EncapDevOutput(netDev, nsPkt.optMacAddr, srcAddr, naPkt.icmpv6NaToPacket(), IPV6_PROTOCOL_NUM_ICMP)
		} else {
			fmt.Printf("ns target not match! targetAddr is %s, ipv6-device is %s\n", targetAddrStr, fmtIpStr(netDev.ipv6Dev.address))
		}
		break
	case ICMPV6_TYPE_NEIGHBOR_ADVERTISEMENT:
		if len(icmpPacket) < 32 {
			fmt.Printf("received icmpv6 NA Packet is too short. size is %d", len(icmpPacket))
			return
		}
		targetAddr := in6Addr(icmpPacket[8:24])
		targetAddrStr := fmtIpStr(targetAddr)
		fmt.Printf("icmpv6 NA packet. targetAddr is %s\n", targetAddrStr)

		naPkt := icmpv6Na{
			hdr:        recIcmpHdr,
			flags:      byteToUint32(icmpPacket[4:8]),
			targetAddr: targetAddr,
			optType:    icmpPacket[24],
			optLength:  icmpPacket[25],
			optMacAddr: [6]uint8(icmpPacket[26:32]), // オプション領域に入るのがアドレス解決の答えになるMACアドレス
		}

		updateNDTableEntry(netDev, naPkt.optMacAddr, naPkt.targetAddr)
		fmt.Printf("updating nd entry %s => %s\n", targetAddrStr, fmtMacStr(naPkt.optMacAddr))

		break
	case ICMPV6_TYPE_ECHO_REQUEST:
		id := byteToUint16(icmpPacket[4:6])
		seq := byteToUint16(icmpPacket[6:8])
		fmt.Printf("received echo request id=%d seq=%d\n", id, seq)

		replyIcmpv6echo := icmpv6Echo{
			header: icmpv6Hdr{
				icmpType: ICMPV6_TYPE_ECHO_REPLY,
				code:     0,
				checksum: 0,
			},
			id:   id,
			seq:  seq,
			data: icmpPacket[8:],
		}

		phdr := ipv6PseudoHeader{
			srcAddr:      netDev.ipv6Dev.address,
			dstAddr:      srcAddr,
			packetLength: uint32(len(icmpPacket)),
			zero:         [3]byte{0, 0, 0},
			nextHeader:   IPV6_PROTOCOL_NUM_ICMP,
		}

		psum := checksum16(phdr.toPseudoHeader(), 0)
		replyIcmpv6echo.header.checksum = checksum16(replyIcmpv6echo.icmpv6EchoToPacket(), psum^0xffff)
		ipv6EncapOutput(srcAddr, netDev.ipv6Dev.address, replyIcmpv6echo.icmpv6EchoToPacket(), IPV6_PROTOCOL_NUM_ICMP)

		break
	}
}

func sendNsPacket(netDev *netDevice, targetAddr in6Addr) {
	// 要請ノードマルチキャストアドレスを生成。近隣要請パケットはマルチキャストに解決したいアドレスの下位3byteを設定したアドレス宛に送信する。
	// ff00::1がマルチキャスト下から２番目からの4bitがフラグ、最下位の4ビットがスコープを表す
	mcastAddr := net.ParseIP(IPV6_MULTICAST_ADDRESS).To16()
	mcastAddr[13] = targetAddr[13]
	mcastAddr[14] = targetAddr[14]
	mcastAddr[15] = targetAddr[15]

	phdr := ipv6PseudoHeader{
		srcAddr:      netDev.ipv6Dev.address,
		dstAddr:      in6Addr(mcastAddr),
		packetLength: uint32(unsafe.Sizeof(icmpv6Na{})), // nits: 直書きでいいかも
		zero:         [3]byte{0x00, 0x00, 0x00},
		nextHeader:   IPV6_PROTOCOL_NUM_ICMP,
	}

	psum := checksum16(phdr.toPseudoHeader(), 0)

	nsPkt := &icmpv6Na{
		hdr: icmpv6Hdr{
			icmpType: ICMPV6_TYPE_NEIGHBOR_SOLICIATION,
			code:     0,
			checksum: 0,
		},
		flags:      0,
		targetAddr: targetAddr,
		optType:    ICMPV6_OPTION_SOURCE_LINK_LAYER_ADDRESS,
		optLength:  1,
		// When sent in response to a multicast Neighbor Solicitation, a Neighbor Advertisement message must contain a Target Link-Layer Address option, which carries the link-layer address of the device sending the message.
		optMacAddr: netDev.macAddr,
	}

	nsPkt.hdr.checksum = checksum16(nsPkt.icmpv6NaToPacket(), psum^0xffff)
	fmt.Printf("sending NS...\n")

	ipv6EncapDevMcastOutput(netDev, targetAddr, nsPkt.icmpv6NaToPacket(), IPV6_PROTOCOL_NUM_ICMP)
}

func (icmpv icmpv6Na) icmpv6NaToPacket() []byte {
	var b bytes.Buffer

	b.Write(uint8ToByte(icmpv.hdr.icmpType))
	b.Write(uint8ToByte(icmpv.hdr.code))
	b.Write(uint16ToByte(icmpv.hdr.checksum))
	b.Write(uint32ToByte(icmpv.flags))
	b.Write(icmpv.targetAddr[:])
	b.Write(uint8ToByte(icmpv.optType))
	b.Write(uint8ToByte(icmpv.optLength))
	b.Write(icmpv.optMacAddr[:])

	return b.Bytes()
}

func (icmpv6echo icmpv6Echo) icmpv6EchoToPacket() []byte {
	var b bytes.Buffer

	b.Write(uint8ToByte(icmpv6echo.header.icmpType))
	b.Write(uint8ToByte(icmpv6echo.header.code))
	b.Write(uint16ToByte(icmpv6echo.header.checksum))
	b.Write(uint16ToByte(icmpv6echo.id))
	b.Write(uint16ToByte(icmpv6echo.seq))
	b.Write(icmpv6echo.data[:])

	return b.Bytes()
}
