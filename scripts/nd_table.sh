#!/bin/sh

HOST1_ROUTER1_IPV6="2001:db8:0:1001::2"
ROUTER1_HOST1_IPV6="2001:db8:0:1001::1"

ROUTER1_HOST1_MAC_ADDR=$(ip netns exec router1 bash -c '
  ip l show dev router1-host1 | grep -oE "([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}" | head -n 1
')
ip netns exec host1 ip -6 neigh add ${ROUTER1_HOST1_IPV6} lladdr ${ROUTER1_HOST1_MAC_ADDR} dev host1-router1
ip netns exec host1 ip -6 neigh

#HOST1_ROUTER1_MAC_ADDR=$(ip netns exec host1 bash -c '
#  ip l show dev host1-router1 | grep -oE "([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}" | head -n 1
#')
#ip netns exec router1 ip -6 neigh add ${HOST1_ROUTER1_IPV6} lladdr ${HOST1_ROUTER1_MAC_ADDR} dev router1-host1
#ip netns exec router1 ip -6 neigh
