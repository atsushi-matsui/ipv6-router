package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func configIpv6NetRoute(prefix in6Addr, prefixLen uint8, nextHop in6Addr) {
	route := &ipv6RouteEntry{
		routeType: NETWORK,
		nextHop:   nextHop,
	}
	patriciaTrieInsert(prefix, prefixLen, route)

	fmt.Printf("configure route to %s/%d via %s\n", fmtIpStr(prefix), prefixLen, fmtIpStr(nextHop))
}

func configIpv6Addr(netDev *netDevice, addr in6Addr, prefixLen uint8) {
	if netDev == nil {
		fmt.Printf("net device to configure not found\n")
		return
	}

	netDev.ipv6Dev.address = addr
	netDev.ipv6Dev.prefixLen = prefixLen

	fmt.Printf("configure ipv6 address to %s\n", fmtIpStr(addr))

	route := &ipv6RouteEntry{
		routeType: CONNECTED,
		dev:       netDev,
	}
	patriciaTrieInsert(addr, prefixLen, route)

	fmt.Printf("configure directly connected route %s/%d. device name is %s\n", fmtIpStr(in6AddrClearPrefix(addr, prefixLen)), prefixLen, netDev.name)
}

func getMacAddr(netns string, ifName string) [6]byte {
	// コマンドと引数を指定
	cmd := exec.Command("ip", "netns", "exec", netns, "bash", "-c", fmt.Sprintf(`
		ip l show dev %s | grep -oE "([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}" | head -n 1
	`, ifName))

	// 標準出力をキャプチャ
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Errorf("error: %s", err)
	}

	macStr := strings.TrimSpace(string(output))
	if macStr == "" {
		fmt.Errorf("invalid mac addr format", err)
	}

	// MACアドレスを[6]byte形式に変換
	return parseMac(macStr)
}
