FROM golang:latest

ARG WORKSPACE="/workspaces/ipv6-router"

RUN apt-get update && \
    apt-get install -y \
        iproute2 \
        ethtool \
        iputils-ping \
        net-tools \
        ndisc6 \
        tcpdump

COPY ./ /workspaces/ipv6-router
WORKDIR ${WORKSPACE}
