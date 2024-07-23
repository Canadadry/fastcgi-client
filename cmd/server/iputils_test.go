package server

import (
	"testing"
)

func TestSplitIPAndPort(t *testing.T) {
	tests := map[string]struct {
		input        string
		expectedIP   string
		expectedPort string
	}{
		"valid ipv4:port":       {"192.168.1.1:8080", "192.168.1.1", "8080"},
		"valid ipv6:port":       {"[2001:db8::1]:8080", "2001:db8::1", "8080"},
		"valid ipv6":            {"2001:db8::1", "2001:db8::1", ""},
		"valid ipv4":            {"127.0.0.1", "127.0.0.1", ""},
		"valid ipv6:empty port": {"[2001:db8::1]:", "2001:db8::1", ""},
		"valid ipv4:empty port": {"127.0.0.1:", "127.0.0.1", ""},
		"valid :port":           {":8080", "", "8080"},
		"invalid ip":            {"192.168.1..1:", "", ""},
		"random string":         {"test:error", "", ""},
		"too big port":          {":123456789", "", ""},
		"negative port":         {":-1", "", ""},
	}

	for _, test := range tests {
		ip, port := splitIPAndPort(test.input)
		if ip != test.expectedIP || port != test.expectedPort {
			t.Errorf("For input '%s', expected IP: '%s', Port: '%s', but got IP: '%s', Port: '%s'", test.input, test.expectedIP, test.expectedPort, ip, port)
		}
	}
}
