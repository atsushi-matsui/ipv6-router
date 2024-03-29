package main

import (
	"bytes"
	"fmt"
	"net"
	"reflect"
)

const IPV6_PROTOCOL_NUM_ICMP uint8 = 0x3a

const ICMPV6_OPTION_SOURCE_LINK_LAYER_ADDRESS uint8 = 1
const ICMPV6_OPTION_TARGET_LINK_LAYER_ADDRESS uint8 = 2

type ipv6RouteType int

const (
	CONNECTED ipv6RouteType = iota
	NETWORK
)

type in6Addr [16]byte

type ipv6Device struct {
	address   in6Addr // IPv6アドレス
	prefixLen uint8   // プレフィックス長(0~128)
	scope     uint8   // スコープ
}

type ipv6Header struct {
	verTcFl    uint32 // Version(4bit) + Traffic Class(8bit) + Flow Label(20bit)
	payloadLen uint16
	nextHdr    uint8
	hopLimit   uint8
	srcAddr    in6Addr
	dstAddr    in6Addr
}

type ipv6PseudoHeader struct {
	srcAddr      in6Addr
	dstAddr      in6Addr
	packetLength uint32
	zero         [3]byte
	nextHeader   uint8
}

type ipv6RouteEntry struct {
	routeType ipv6RouteType
	dev       *netDevice
	nextHop   in6Addr
}

func newIpv6(addr in6Addr, prefixLen uint8) *ipv6Device {
	return &ipv6Device{
		address:   addr,
		prefixLen: prefixLen,
	}
}

func ipv6Input(netDev *netDevice, buffer []byte) {
	if netDev.ipv6Dev == nil {
		fmt.Printf("received ipv6 packet from non ipv6 device %s\n", netDev.name)
		return
	}
	if len(buffer) < 40 {
		fmt.Printf("received ipv6 packet too short from %s\n", netDev.name)
		return
	}

	ipv6header := ipv6Header{
		verTcFl:    byteToUint32(buffer[0:4]),
		payloadLen: byteToUint16(buffer[4:6]),
		nextHdr:    buffer[6],
		hopLimit:   buffer[7],
		srcAddr:    in6Addr(buffer[8:24]),
		dstAddr:    in6Addr(buffer[24:40]),
	}

	version := (ipv6header.verTcFl >> 28) & 0x0F
	if version != 6 {
		fmt.Printf("ip header version is %d\n", version)
		return
	}

	// マルチキャストアドレスの判定
	if ipv6header.dstAddr[0] == 0xff { // ff00::/8の範囲だったら
		if reflect.DeepEqual(netDev.ipv6Dev.address[13:16], ipv6header.dstAddr[13:16]) {
			fmt.Printf("multicast. ip is %s\n", fmtIpStr(ipv6header.dstAddr))
			ipv6InputToOurs(netDev, &ipv6header, buffer[40:])
			return
		}
	}

	// 宛先IPアドレスをルータが持ってるか調べる
	for _, netDevice := range netDevices {
		if netDevice.ipv6Dev.address == ipv6header.dstAddr {
			fmt.Printf("router know ip. device ip is %s\n", fmtIpStr(ipv6header.dstAddr))
			ipv6InputToOurs(netDev, &ipv6header, buffer[40:])
			return
		}
	}

	// 宛先IPアドレスがルータの持っているIPアドレスでない場合はフォワーディングを行う
	fmt.Printf("start forwarding!\n")
	resNode := patriciaTrieSearch(ipv6header.dstAddr)

	if resNode == nil {
		fmt.Printf("no route to %s\n", fmtIpStr(ipv6header.dstAddr))
		return
	}

	ipv6header.hopLimit--

	outedPacket := ipv6header.toPacket()
	outedPacket = append(outedPacket, buffer[40:]...)

	switch resNode.route.routeType {
	case CONNECTED:
		fmt.Printf("forwarding ipv6 packet to host\n")
		ipv6OutputToHost(resNode.route.dev, ipv6header.dstAddr, ipv6header.srcAddr, outedPacket)
	case NETWORK:
		fmt.Printf("forwarding ipv6 packet to network\n")
		ipv6OutputToNextHop(resNode.route.nextHop, outedPacket)
	}
}

func ipv6InputToOurs(netDev *netDevice, ipv6header *ipv6Header, buffer []byte) {
	switch ipv6header.nextHdr {
	case IPV6_PROTOCOL_NUM_ICMP:
		icmpv6Input(netDev, ipv6header.srcAddr, ipv6header.dstAddr, buffer)
	default:
		fmt.Printf("unhandled next header : %d\n", ipv6header.nextHdr)
	}
}

func ipv6EncapOutput(dstAddr in6Addr, srcAddr in6Addr, buffer []byte, nextHdrNum uint8) {
	ipv6header := ipv6Header{
		verTcFl:    0x60000000,
		payloadLen: uint16(len(buffer)),
		nextHdr:    nextHdrNum,
		hopLimit:   0xff,
		srcAddr:    srcAddr,
		dstAddr:    dstAddr,
	}
	packet := ipv6header.toPacket()
	packet = append(packet, buffer...)

	// ルーティング/フォワーディングが実装されてないとき用
	//for _, dev := range netDevices {
	//	if dev.ipv6Dev == nil {
	//		continue
	//	}
	//	// 宛先アドレスと同じネットワークを持ったデバイスを探す
	//	if in6IsInNetwork(dstAddr, dev.ipv6Dev.address, int(dev.ipv6Dev.prefixLen)) {
	//		var packet []byte
	//		packet = ipv6header.toPacket()
	//		packet = append(packet, buffer...)
	//		ipv6OutputToHost(dev, dstAddr, srcAddr, packet)
	//
	//		return
	//	}
	//}

	resNode := patriciaTrieSearch(dstAddr)

	if resNode != nil && resNode.route != nil {
		switch resNode.route.routeType {
		case CONNECTED:
			ipv6OutputToHost(resNode.route.dev, dstAddr, resNode.route.dev.ipv6Dev.address, packet)
		case NETWORK:
			ipv6OutputToNextHop(resNode.route.nextHop, packet)
		}
	} else {
		fmt.Printf("not found dst addr in forwarding table. addr is %s\n", fmtIpStr(dstAddr))
	}
}

func in6IsInNetwork(address in6Addr, prefix in6Addr, prefixLen int) bool {
	for i := 0; i < prefixLen; i++ {
		byteIndex := i / 8
		bitIndex := uint(i % 8)

		if address[byteIndex]&(1<<(7-bitIndex)) != prefix[byteIndex]&(1<<(7-bitIndex)) {
			return false
		}
	}

	return true
}

/*
 * ipv6ではNS/NAを利用したアドレス解決を利用。NSはARPリクエストに相当し、NAはARPリプライに相当。
 */
func ipv6OutputToHost(netDev *netDevice, dstAddr in6Addr, srcAddr in6Addr, buffer []byte) {
	nde := searchNDTableEntry(dstAddr)
	if nde == nil { // ARPエントリが無かったら
		fmt.Printf("trying ipv6 output to host, but no nd record to %s\n", fmtIpStr(dstAddr))
		sendNsPacket(netDev, dstAddr)
	} else {
		// イーサネットでカプセル化して送信
		fmt.Printf("trying ipv6 output to host, find nd record to %s\n", fmtIpStr(dstAddr))
		ethernetEncapsulateOutput(nde.dev, nde.macAddr, buffer, ETHER_TYPE_IPV6)
	}
}

func ipv6OutputToNextHop(dstAddr in6Addr, buffer []byte) {
	ndTableEntry := searchNDTableEntry(dstAddr)

	if ndTableEntry == nil {
		resNode := patriciaTrieSearch(dstAddr)

		if resNode != nil && resNode.route != nil {
			if resNode.route.routeType == CONNECTED {
				sendNsPacket(resNode.route.dev, dstAddr)
				return
			}
		}
	} else {
		fmt.Printf("found nd entry to next hop!\n")
		ethernetEncapsulateOutput(ndTableEntry.dev, ndTableEntry.macAddr, buffer, ETHER_TYPE_IPV6)
	}
}

func ipv6EncapDevOutput(netDev *netDevice, dstMacAddr [6]uint8, dstAddr in6Addr, buffer []byte, nextHdrNum uint8) {
	var v6hMybuf []byte

	ipv6hdr := ipv6Header{
		verTcFl:    0x60000000,
		payloadLen: uint16(len(buffer)),
		nextHdr:    nextHdrNum,
		hopLimit:   0xff,
		srcAddr:    netDev.ipv6Dev.address,
		dstAddr:    dstAddr,
	}

	v6hMybuf = append(v6hMybuf, ipv6hdr.toPacket()...)
	v6hMybuf = append(v6hMybuf, buffer...)

	ethernetEncapsulateOutput(netDev, dstMacAddr, v6hMybuf, ETHER_TYPE_IPV6)
}

func ipv6EncapDevMcastOutput(netDev *netDevice, dstAddr in6Addr, buffer []byte, nextHdrNum uint8) {
	var v6hMybuf []byte

	ipv6hdr := ipv6Header{
		verTcFl:    0x60000000,
		payloadLen: uint16(len(buffer)),
		nextHdr:    nextHdrNum,
		hopLimit:   0xff,
		srcAddr:    netDev.ipv6Dev.address,
		dstAddr:    dstAddr,
	}

	v6hMybuf = append(v6hMybuf, ipv6hdr.toPacket()...)
	v6hMybuf = append(v6hMybuf, buffer...)

	// マルチキャストアドレスを指定
	var dstMacAddr [6]uint8
	dstMacAddr[0] = 0x33
	dstMacAddr[1] = 0x33
	copy(dstMacAddr[2:6], dstAddr[12:16])

	ethernetEncapsulateOutput(netDev, dstMacAddr, v6hMybuf, ETHER_TYPE_IPV6)
}

func (ipv6header ipv6Header) toPacket() []byte {
	var b bytes.Buffer

	b.Write(uint32ToByte(ipv6header.verTcFl))
	b.Write(uint16ToByte(ipv6header.payloadLen))
	b.Write(uint8ToByte(ipv6header.nextHdr))
	b.Write(uint8ToByte(ipv6header.hopLimit))
	b.Write(ipv6header.srcAddr[:])
	b.Write(ipv6header.dstAddr[:])

	return b.Bytes()
}

func (pHdr ipv6PseudoHeader) toPseudoHeader() []byte {
	var b bytes.Buffer

	b.Write(pHdr.srcAddr[:])
	b.Write(pHdr.dstAddr[:])
	b.Write(uint32ToByte(pHdr.packetLength))
	b.Write(pHdr.zero[:])
	b.Write(uint8ToByte(pHdr.nextHeader))

	return b.Bytes()
}

func getIpv6Device(addrs []net.Addr) *in6Addr {
	for _, addr := range addrs {

		ip, _, err := net.ParseCIDR(addr.String())
		if err != nil {
			fmt.Printf("Error parsing address %s: %v\n", addr, err)
			continue
		}
		// IPv6アドレスの場合のみ処理
		if ip.To4() == nil && ip.To16() != nil {
			ipv6Addr := ip.To16()
			result := &in6Addr{}
			copy(result[:], ipv6Addr)
			return result
		}
	}

	return nil
}

func in6AddrSum(addr in6Addr) uint32 {
	var result uint32
	for i := 0; i < len(addr); i += 4 {
		segment := byteToUint32(addr[i : i+4])
		result |= segment
	}
	return result
}
