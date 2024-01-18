package main

import "fmt"

/**
 * IPv6ルーティングテーブルのルートノード
 */
var ipv6Fib *patriciaNode

type patriciaNode struct {
	left     *patriciaNode
	right    *patriciaNode
	parent   *patriciaNode
	addr     in6Addr
	bitsLen  uint8 // このノードで比較するビットの位置
	isPrefix bool  // このノードがプレフィックスを表すかどうか
	route    *ipv6RouteEntry
}

func in6AddrGetBit(address in6Addr, bit uint8) uint8 {
	if bit > 128 {
		fmt.Errorf("bit length is within 128. actual length is %d\n", bit)
	}
	byteIndex := bit / 8
	bitIndex := 7 - (bit % 8)
	return (address[byteIndex] >> bitIndex) & 0b01
}

func in6AddrClearBit(address *in6Addr, bit uint8) {
	if bit < 0 || bit >= 128 {
		panic("Invalid bit index")
	}
	byteIndex := bit / 8
	bitIndex := 7 - (bit % 8)
	address[byteIndex] &= ^(1 << uint(bitIndex))
}

func in6AddrGetMatchBitsLen(addr1, addr2 in6Addr, endBit uint8) uint8 {
	var startBit uint8 = 0
	if startBit > endBit {
		panic("Invalid start and end bit indices")
	}
	if endBit >= 128 {
		panic("Invalid end bit index")
	}

	var count uint8 = 0
	for i := startBit; i <= endBit; i++ {
		if in6AddrGetBit(addr1, i) != in6AddrGetBit(addr2, i) {
			return count
		}
		count++
	}
	return count
}

func in6AddrClearPrefix(addr in6Addr, prefixLen uint8) in6Addr {
	for i := prefixLen; i < 128; i++ {
		in6AddrClearBit(&addr, i)
	}
	return addr
}

func createPatriciaNode(addr in6Addr, bitsLen uint8, isPrefix bool, parent *patriciaNode) *patriciaNode {
	node := &patriciaNode{
		parent:   parent,
		addr:     addr,
		bitsLen:  bitsLen,
		isPrefix: isPrefix,
	}

	return node
}

func patriciaTrieInsert(address in6Addr, prefixLen uint8, route *ipv6RouteEntry) {
	var currentBitsLen uint8 = 0
	currentNode := ipv6Fib
	var nextNode *patriciaNode

	// 引数で渡されたプレフィックスをきれいにする
	address = in6AddrClearPrefix(address, prefixLen)

	// 枝を辿る
	for {
		if in6AddrGetBit(address, currentBitsLen) == 0 {
			nextNode = currentNode.left
			if nextNode == nil {
				// ノードを作成
				currentNode.left = createPatriciaNode(address, prefixLen-currentBitsLen, true, currentNode)
				currentNode.left.route = route
				break
			}
		} else {
			nextNode = currentNode.right
			if nextNode == nil {
				// ノードを作成
				currentNode.right = createPatriciaNode(address, prefixLen-currentBitsLen, true, currentNode)
				currentNode.right.route = route
				break
			}
		}

		matchLen := in6AddrGetMatchBitsLen(address, nextNode.addr, currentBitsLen+nextNode.bitsLen-1)

		if matchLen == currentBitsLen+nextNode.bitsLen {
			// 次のノードと全マッチ
			currentBitsLen += nextNode.bitsLen
			currentNode = nextNode

			if currentBitsLen == prefixLen {
				// 目標だった時
				nextNode.isPrefix = true
				nextNode.route = route
				break
			}

		} else { // 途中までは一致している
			// 中間nodeを作成して、currentNodeは親となる。
			midBitLen := matchLen - currentBitsLen
			midAddress := in6AddrClearPrefix(address, matchLen)
			midNode := createPatriciaNode(midAddress, midBitLen, false, currentNode)

			if currentNode.left == nextNode {
				// Current-Intermediateをつなぎなおす
				currentNode.left = midNode
			} else {
				currentNode.right = midNode
			}

			nextNode.bitsLen -= midBitLen
			nextNode.parent = midNode

			fmt.Printf("Separated %d & %d\n", midBitLen, nextNode.bitsLen)

			//if prefixLen == currentBitsLen+matchLen {
			//	nextNode.isPrefix = true
			//	nextNode.route = route
			//	break
			//}

			diffNextToMidLen := prefixLen - matchLen
			diffNextToMidNode := createPatriciaNode(address, diffNextToMidLen, true, midNode)
			diffNextToMidNode.route = route
			if in6AddrGetBit(address, matchLen) == 0 {
				// Intermediate-Nextをつなぎなおす&目的のノードを作る
				midNode.right = nextNode
				midNode.left = diffNextToMidNode
			} else {
				midNode.left = nextNode
				midNode.right = diffNextToMidNode
			}

			break
		}
	}
}

func patriciaTrieSearch(address in6Addr) *patriciaNode {
	var currentBitsLen uint8 = 0
	currentNode := ipv6Fib
	var nextNode *patriciaNode
	var lastMatched *patriciaNode

	for currentBitsLen < 128 { // 最後までたどり着いてない間は進める
		// 進めるノードの選択
		if in6AddrGetBit(address, currentBitsLen) == 0 {
			nextNode = currentNode.left
		} else {
			nextNode = currentNode.right
		}

		if nextNode == nil {
			break
		}

		matchLen := in6AddrGetMatchBitsLen(address, nextNode.addr, currentBitsLen+nextNode.bitsLen-1)

		if nextNode.isPrefix {
			lastMatched = nextNode
		}

		if matchLen != currentBitsLen+nextNode.bitsLen {
			break
		}

		currentNode = nextNode
		currentBitsLen += nextNode.bitsLen
	}

	if lastMatched != nil && lastMatched.route != nil {
		switch lastMatched.route.routeType {
		case CONNECTED:
			fmt.Printf("find to host node. address id %s\n", fmtIpStr(lastMatched.route.dev.ipv6Dev.address))
		case NETWORK:
			fmt.Printf("find to next hop node. address id %s\n", fmtIpStr(lastMatched.route.nextHop))
		}
	}

	return lastMatched
}
