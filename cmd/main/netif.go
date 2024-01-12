package main

import (
	"fmt"
	"net"
	"syscall"
)

var ignoreIfNames = []string{"lo", "bond0", "dummy0", "tunl0", "sit0"}

type netDevice struct {
	name      string // インターフェース名
	macAddr   [6]uint8
	socketFd  int
	sockAddr  syscall.SockaddrLinklayer
	ethHeader *ethernetHeader
	ipv6Dev   *ipv6Device
}

func newNetIf(
	name string,
	macAddr net.HardwareAddr,
	socketFd int,
	sockAddr syscall.SockaddrLinklayer,
	ipv6Dev *ipv6Device) *netDevice {
	return &netDevice{
		name:     name,
		macAddr:  setMacAddr(macAddr),
		socketFd: socketFd,
		sockAddr: sockAddr,
		ipv6Dev:  ipv6Dev,
	}
}

/* ネットデバイスの送信処理 */
func (netDev netDevice) transmit(buffer []uint8) error {
	err := syscall.Sendto(netDev.socketFd, buffer, 0, &netDev.sockAddr)
	if err != nil {
		return err
	}
	return nil
}

/* ネットワークデバイスの受信処理 */
func (netDev *netDevice) poll() error {
	recvBuffer := make([]byte, 1500)
	n, _, err := syscall.Recvfrom(netDev.socketFd, recvBuffer, 0)
	if err != nil {
		if n == -1 {
			return nil
		} else {
			return fmt.Errorf("recv err, n is %d, device is %s, err is %s", n, netDev.name, err)
		}
	}

	// 受信したデータを表示してみる
	fmt.Printf("Received %d bytes from %s: %x\n", n, netDev.name, recvBuffer[:n])

	// 受信したデータをイーサネットに送る
	ethernetInput(netDev, recvBuffer[:n])

	return nil
}

func (netDev *netDevice) setEthHeader(ethHeader *ethernetHeader) {
	netDev.ethHeader = ethHeader
}

// htons converts a short (uint16) from host-to-network byte order.
func htons(i uint16) uint16 {
	return (i<<8)&0xff00 | i>>8
}

func isIgnoreIf(name string) bool {
	for _, ignoreIfName := range ignoreIfNames {
		if name == ignoreIfName {
			return true
		}
	}

	return false
}
