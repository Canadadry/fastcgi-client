package server

import (
	"net"
	"strconv"
	"strings"
)

const maxPort = int(^uint16(0))

func splitIPAndPort(address string) (string, string) {
	lastColon := strings.LastIndex(address, ":")
	if lastColon == -1 {
		if net.ParseIP(address) != nil {
			return address, ""
		}
		return "", ""
	}

	ipPart := address[:lastColon]
	if net.ParseIP(ipPart) == nil {
		ipPart = ""
	}

	portPart := address[lastColon+1:]
	if port, err := strconv.Atoi(portPart); err != nil || port > maxPort || port <= 0 {
		portPart = ""
	}

	return ipPart, portPart
}
