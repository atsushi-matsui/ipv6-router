package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
)

func byteToUint16(b []byte) uint16 {
	return binary.BigEndian.Uint16(b)
}

func byteToUint32(b []byte) uint32 {
	return binary.BigEndian.Uint32(b)
}

func uint8ToByte(i uint8) []byte {
	return []byte{i}
}

func uint16ToByte(i uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, i)
	return b
}

func uint32ToByte(i uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, i)
	return b
}

func checksum16(buffer []byte, start uint16) uint16 {
	sum := uint32(start)

	// まず16ビット毎に足す
	for i := 0; i < len(buffer)-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(buffer[i : i+2]))
	}

	// もし1バイト余ってたら足す
	if len(buffer)%2 != 0 {
		sum += uint32(buffer[len(buffer)-1])
	}

	// あふれた桁を折り返して足す
	for sum>>16 > 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}

	return uint16(^sum)
}

func fmtMacStr(macAddr [6]byte) string {
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", macAddr[0], macAddr[1], macAddr[2], macAddr[3], macAddr[4], macAddr[5])
}

func fmtIpStr(ipAddr in6Addr) string {
	return net.IP(ipAddr[:]).String()
}

func parseIpv6(ipStr string) in6Addr {
	return in6Addr(net.ParseIP(ipStr).To16())
}

func parseMac(macStr string) [6]byte {
	macAddr, err := hex.DecodeString(strings.ReplaceAll(macStr, ":", ""))
	if err != nil {
		fmt.Errorf("mac addr parse error. macStr is %d", macStr)
	}

	var result [6]byte
	copy(result[:], macAddr)
	return result
}
