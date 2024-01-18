package main

import (
	"bytes"
	"fmt"
	"log"
)

const ETHER_TYPE_IPV6 uint16 = 0x86dd

var ETHER_ADDR_IPV6_MCAST_PREFIX = [2]byte{0x33, 0x33}
var ETHERNET_ADDRESS_BROADCAST = [6]uint8{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

type ethernetHeader struct {
	dstAddr [6]uint8
	srcAddr [6]uint8
	ethType uint16
}

func newEth(dstAddr []byte, srcAddr []byte, ethType []byte) *ethernetHeader {
	return &ethernetHeader{
		dstAddr: setMacAddr(dstAddr),
		srcAddr: setMacAddr(srcAddr),
		ethType: byteToUint16(ethType),
	}
}

func (ethHeader ethernetHeader) toEthPacket() []byte {
	var b bytes.Buffer
	b.Write(ethHeader.dstAddr[:])
	b.Write(ethHeader.srcAddr[:])
	b.Write(uint16ToByte(ethHeader.ethType))

	return b.Bytes()
}

func setMacAddr(macAddrByte []byte) [6]uint8 {
	macAddrLen := len(macAddrByte)
	if macAddrLen != 6 {
		log.Fatalf("invalid mac address length is %d, address is %v", macAddrLen, macAddrByte)
	}
	var macAddrUint8 [6]uint8
	for i, v := range macAddrByte {
		macAddrUint8[i] = v
	}
	return macAddrUint8
}

func ethernetInput(netDev *netDevice, buffer []byte) {
	netDev.setEthHeader(newEth(buffer[0:6], buffer[6:12], buffer[12:14]))
	dstAddrStr := fmtMacStr(netDev.ethHeader.dstAddr)
	deviceAddrStr := fmtMacStr(netDev.macAddr)

	// 自分のMACアドレス宛てかマルチキャストアドレスでなければ終了する
	if netDev.ethHeader.dstAddr != netDev.macAddr && netDev.ethHeader.dstAddr != ETHERNET_ADDRESS_BROADCAST && !(buffer[0] == ETHER_ADDR_IPV6_MCAST_PREFIX[0] && buffer[1] == ETHER_ADDR_IPV6_MCAST_PREFIX[1]) {
		fmt.Printf("not handle address, dstMacAddr is %s, device addr is %s\n", dstAddrStr, deviceAddrStr)
		return
	}

	switch netDev.ethHeader.ethType {
	case ETHER_TYPE_IPV6:
		fmt.Printf("ether type is ipv6, netDev is %+v, dstMacAddr is %s\n", netDev, dstAddrStr)
		// Etherフレームヘッダ（宛先MAC：6byte、送信先MAC：6byte、タイプ：2byte）を除くパケットを渡す
		ipv6Input(netDev, buffer[14:])
	default:
		fmt.Printf("not handle ether type, type is %d, netDev is %+v\n", netDev.ethHeader.ethType, netDev)
	}
}

func ethernetEncapsulateOutput(netDev *netDevice, dstAddr [6]uint8, buffer []byte, etherType uint16) {
	fmt.Printf("sending ethernet frame type %04x from %s to %s\n", etherType, fmtMacStr(netDev.macAddr), fmtMacStr(dstAddr))

	ethHeaderPacket := ethernetHeader{
		dstAddr: dstAddr,
		srcAddr: netDev.macAddr,
		ethType: etherType,
	}.toEthPacket()

	err := netDev.transmit(append(ethHeaderPacket, buffer...))
	if err != nil {
		log.Fatalf("transmit is err : %v", err)
	}
}
