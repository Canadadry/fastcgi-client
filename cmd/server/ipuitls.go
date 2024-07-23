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
		if isIPValid(address) {
			return address, ""
		}
		return "", ""
	}
	firstColon := strings.Index(address, ":")
	if firstColon == lastColon {
		return splitIPv4AndPort(address, lastColon)
	}
	return splitIPv6AndPort(address, lastColon)
}

func isIPValid(address string) bool {
	return net.ParseIP(address) != nil
}

func splitIPv4AndPort(address string, lastColon int) (string, string) {
	ipPart := address[:lastColon]
	if !isIPValid(ipPart) {
		ipPart = ""
	}

	portPart := address[lastColon+1:]
	if port, err := strconv.Atoi(portPart); err != nil || port > maxPort || port <= 0 {
		portPart = ""
	}

	return ipPart, portPart
}

func splitIPv6AndPort(address string, lastColon int) (string, string) {
	if address[0] != '[' {
		if isIPValid(address) {
			return address, ""
		}
		return "", ""
	}

	ipPart := address[1 : lastColon-1]
	if !isIPValid(ipPart) {
		ipPart = ""
	}

	portPart := address[lastColon+1:]
	if port, err := strconv.Atoi(portPart); err != nil || port > maxPort || port <= 0 {
		portPart = ""
	}

	return ipPart, portPart
}
