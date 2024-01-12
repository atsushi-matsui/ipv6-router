package main

import (
	"bytes"
	"fmt"
)

const ND_TABLE_SIZE = 1111

var ndTable map[uint32]*ndTableEntry

type ndTableEntry struct {
	macAddr [6]uint8
	v6Addr  in6Addr
	dev     *netDevice
	next    *ndTableEntry
}

func initNdTable() {
	ndTable = make(map[uint32]*ndTableEntry)
}

func updateNDTableEntry(netDev *netDevice, macAddr [6]uint8, v6Addr in6Addr) {
	candidate := ndTable[in6AddrSum(v6Addr)%ND_TABLE_SIZE]
	macStr := fmtMacStr(macAddr)
	ipv6Str := fmtIpStr(v6Addr)

	for candidate != nil {
		if v6Addr == candidate.v6Addr {
			candidate.macAddr = macAddr
			candidate.v6Addr = v6Addr
			candidate.dev = netDev

			fmt.Printf("update ND table. macAddr is %s, ipAddr is %s\n", macStr, ipv6Str)
			return
		}
		candidate = candidate.next
	}

	// 新規追加
	ndTable[in6AddrSum(v6Addr)%ND_TABLE_SIZE] = &ndTableEntry{
		macAddr: macAddr,
		v6Addr:  v6Addr,
		dev:     netDev,
	}
	fmt.Printf("insert ND table. macAddr is %s, ipAddr is %s\n", macStr, ipv6Str)
}

func searchNDTableEntry(v6Addr in6Addr) *ndTableEntry {
	candidate := ndTable[in6AddrSum(v6Addr)%ND_TABLE_SIZE]

	for candidate != nil {
		if bytes.Equal(v6Addr[:], candidate.v6Addr[:]) {
			return candidate
		}
		candidate = candidate.next
	}

	return nil
}
