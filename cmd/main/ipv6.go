package main

import "net"

type in6Addr [16]byte

type Ipv6Device struct {
	address   in6Addr // IPv6アドレス
	prefixLen uint32  // プレフィックス長(0~128)
	scope     uint8   // スコープ
	// netDev *netDevice; // ネットワークデバイスへのポインタ
}

func newIpv6(addressStr string, prefixLen uint32) *Ipv6Device {
	return &Ipv6Device{
		address:   in6Addr(net.ParseIP(addressStr)),
		prefixLen: prefixLen,
	}
}
