package main

type patriciaNode struct {
	left     *patriciaNode
	right    *patriciaNode
	parent   *patriciaNode
	address  in6Addr
	bitsLen  int // このノードで比較するビットの位置
	isPrefix int // このノードがプレフィックスを表すかどうか
	data     []byte
}

func createPatriciaNode(address in6Addr, bitsLen int, isPrefix bool, parent *patriciaNode) {
	//TODO: 未実装
}

func patriciaTrieSearch(root *patriciaNode, address in6Addr) {
	//TODO: 未実装
}
