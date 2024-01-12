#!/bin/sh

# 仮想のネットワーク空間を作成
ip netns add host1
ip netns add router1
ip netns add router2
ip netns add host2

# veth peerはnetns間の仮想的なNICインターフェイスを提供する
ip link add name host1-router1 type veth peer name router1-host1
ip link add name router1-router2 type veth peer name router2-router1
ip link add name router2-host2 type veth peer name host2-router2

# vethをnetnsに配置する
ip link set host1-router1 netns host1
ip link set router1-host1 netns router1
ip link set router1-router2 netns router1
ip link set router2-router1 netns router2
ip link set router2-host2 netns router2
ip link set host2-router2 netns host2

### host1の設定
# 指定されたnetns内でvethにipv6を付与する
ip netns exec host1 ip addr add 2001:db8:0:1001::2/64 dev host1-router1
# vethを有効化する
ip netns exec host1 ip link set host1-router1 up
# vethの受信(rx)機能と送信機能(tx)をオフにする
ip netns exec host1 ethtool -K host1-router1 rx off tx off
# IPv6デフォルトゲートウェイを設定。"2001:db8:0:1001::1"はゲートウェイのIPv6アドレス
ip netns exec host1 ip route add default via 2001:db8:0:1001::1

### router1の設定
#ip netns exec router1 ip addr add 2001:db8:0:1001::1/64 dev router1-host1
ip netns exec router1 ip link set router1-host1 up
ip netns exec router1 ethtool -K router1-host1 rx off tx off
ip netns exec router1 ip link set router1-router2 up
ip netns exec router1 ethtool -K router1-router2 rx off tx off

### router2の設定
ip netns exec router2 ip addr add 2001:db8:0:1000::2/64 dev router2-router1
ip netns exec router2 ip link set router2-router1 up
ip netns exec router2 ethtool -K router2-router1 rx off tx off
ip netns exec router2 ip addr add 2001:db8:0:1002::1/64 dev router2-host2
ip netns exec router2 ip link set router2-host2 up
ip netns exec router2 ethtool -K router2-host2 rx off tx off
# IPv6パケットの転送を有効にする
ip netns exec router2 sysctl -w net.ipv6.conf.all.forwarding=1
# "2001:db8:0:1001::/64"の次の経路は"2001:db8:0:1000:1"
ip netns exec router2 ip route add 2001:db8:0:1001::/64 via 2001:db8:0:1000::1

### host2の設定
ip netns exec host2 ip addr add 2001:db8:0:1002::2/64 dev host2-router2
ip netns exec host2 ip link set host2-router2 up
ip netns exec host2 ethtool -K host2-router2 rx off tx off
ip netns exec host2 ip route add 2001:db8:0:1001::/64 via 2001:db8:0:1002::1
ip netns exec host2 ip route add 2001:db8:0:1000::/64 via 2001:db8:0:1002::1
