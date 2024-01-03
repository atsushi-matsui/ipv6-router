package main

import (
	"fmt"
	"log"
	"net"
	"syscall"
)

var netDevices []*netDevice

// TODO: なんのやつかわかっていない
var myMacAddress = "2001:db8:0:1001::1"

func main() {

	// インターフェイスを取得する
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Fatalf("Not found interfaces : %s", err)
	}
	fmt.Printf("interfaces: %+v\n", interfaces)

	// epollを作成する
	events := make([]syscall.EpollEvent, 10)
	epollFd, err := syscall.EpollCreate1(0)
	if err != nil {
		log.Fatalf("Failed create epoll : %s", err)
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

		fmt.Printf("socket file descriptor: %+v\n", sockFd)

		// ソケットをインターフェイスにbindする
		socketAddr := syscall.SockaddrLinklayer{
			Protocol: htons(syscall.ETH_P_ALL),
			Ifindex:  inf.Index,
		}
		err = syscall.Bind(sockFd, &socketAddr)
		if err != nil {
			log.Fatalf("Failed bind socket: %s\n", err)
		}

		// epollにソケットを監視させる
		err = syscall.EpollCtl(epollFd, syscall.EPOLL_CTL_ADD, sockFd, &syscall.EpollEvent{
			Events: syscall.EPOLLIN,
			Fd:     int32(sockFd),
		})
		if err != nil {
			log.Fatalf("Failed epollCtl: %s\n", err)
		}

		ipv6Dev := newIpv6(myMacAddress, 64)
		netDev := newNetIf(inf.Name, inf.HardwareAddr, sockFd, socketAddr, ipv6Dev)

		netDevices = append(netDevices, netDev)
		fmt.Printf("Effective netDevice, name is %s, socketFd is %s\n ", netDev.name, netDev.socketFd)
	}

	// 有効なインターフェイスがなければ処理を終了する
	if len(netDevices) == 0 {
		log.Fatalf("No interface is enabled!")
	}

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
