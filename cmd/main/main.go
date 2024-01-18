package main

import (
	"fmt"
	"log"
	"net"
	"syscall"
)

// ネットワーク内のNICのリスト
var netDevices []*netDevice

func main() {

	// インターフェイスを取得する
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Fatalf("Not found interfaces : %s\n", err)
	}
	fmt.Printf("interfaces: %+v\n", interfaces)

	// epollを作成する
	events := make([]syscall.EpollEvent, 10)
	epollFd, err := syscall.EpollCreate1(0)
	if err != nil {
		log.Fatalf("Failed create epoll : %s\n", err)
	}

	for _, inf := range interfaces {
		// 必要なもの以外無視する
		if isIgnoreIf(inf.Name) {
			continue
		}

		// ソケットAPIをオープンする
		protocol := htons(syscall.ETH_P_ALL)
		sockFd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(protocol))
		if err != nil {
			log.Fatalf("Failed open socket: %s\n", err)
		}

		fmt.Printf("socket file descriptor: %d\n", sockFd)

		// ソケットをインターフェイスにbindする
		socketAddr := syscall.SockaddrLinklayer{
			Protocol: htons(syscall.ETH_P_ALL),
			Ifindex:  inf.Index,
		}
		err = syscall.Bind(sockFd, &socketAddr)
		if err != nil {
			log.Fatalf("Failed bind socket: %s\n", err)
		}
		fmt.Printf("bind nic: %+v\n", inf)

		// epollにソケットを監視させる
		err = syscall.EpollCtl(epollFd, syscall.EPOLL_CTL_ADD, sockFd, &syscall.EpollEvent{
			Events: syscall.EPOLLIN,
			Fd:     int32(sockFd),
		})
		if err != nil {
			log.Fatalf("Failed epollCtl: %s\n", err)
		}

		netAddrs, err := inf.Addrs()
		if err != nil {
			log.Fatalf("get ip addr from nic interface is err : %s\n", err)
		}

		ipv6Addr := getIpv6Device(netAddrs)
		if ipv6Addr == nil {
			continue
		}

		ipv6Dev := newIpv6(*ipv6Addr, 64)
		netDev := newNetIf(inf.Name, inf.HardwareAddr, sockFd, socketAddr, ipv6Dev)

		netDevices = append(netDevices, netDev)
		fmt.Printf("effective netDevice, name is %s, socketFd is %d\n", netDev.name, netDev.socketFd)
	}

	// 有効なインターフェイスがなければ処理を終了する
	if len(netDevices) == 0 {
		log.Fatalf("No interface is enabled!\n")
	}

	// ndテーブルの初期化
	initNdTable()

	createPatriciaNode(in6Addr{0x00}, 0, false, nil)

	// ネットワーク設定の投入
	configure()

	// epollでソケットの受信状況を確認する
	for {
		// epoll_waitでパケットの受信を待つ
		nfDs, err := syscall.EpollWait(epollFd, events, -1)
		if err != nil {
			log.Fatalf("epoll wait err : %s", err)
		}
		for i := 0; i < nfDs; i++ {
			// デバイスから通信を受信
			for _, netDev := range netDevices {
				// イベントがあったソケットとマッチしたらパケットを読み込む処理を実行
				if events[i].Fd == int32(netDev.socketFd) {
					err := netDev.poll()
					if err != nil {
						log.Fatal(err)
					}
				}
			}
		}
	}
}

func getNetDevByName(name string) *netDevice {
	for _, netDev := range netDevices {
		if netDev.name == name {
			return netDev
		}
	}

	return nil
}

func configure() {
	ipv6Fib = createPatriciaNode(in6Addr{}, 0, false, nil)

	configIpv6Addr(getNetDevByName("router1-host1"), parseIpv6("2001:db8:0:1001::1"), 64)
	configIpv6Addr(getNetDevByName("router1-router2"), parseIpv6("2001:db8:0:1000::1"), 64)

	configIpv6NetRoute(parseIpv6("2001:db8:0:1002::"), 64, parseIpv6("2001:db8:0:1000::2"))

	updateNDTableEntry(getNetDevByName("router1-host1"), getMacAddr("host1", "host1-router1"), parseIpv6("2001:db8:0:1001::2"))
}
