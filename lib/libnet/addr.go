package libnet

import (
	"net"
	"strings"
)

// StripPort takes an address like "example.com:8080", "127.0.0.1",
// "[2001:db8::1]:443", or even "example.com" (no port), and returns:
//
//	host (string), port (string), err (error)
func StripPort(addr string) (string, string, error) {
	// If the string contains any colon but no explicit port, SplitHostPort fails.
	// To distinguish IPv6 without port, add a dummy port and strip it later.
	if strings.Count(addr, ":") > 1 && !strings.Contains(addr, "]:") {
		// IPv6 literal without brackets — normalize
		addr = "[" + addr + "]"
	}

	host, port, err := net.SplitHostPort(addr)
	if err == nil {
		return host, port, nil
	}

	// If there's no port, SplitHostPort returns an error.
	// Detect that case and return the input unchanged.
	if strings.Contains(err.Error(), "missing port in address") {
		// Strip brackets if present
		host = strings.TrimPrefix(addr, "[")
		host = strings.TrimSuffix(host, "]")
		return host, "", nil
	}

	return "", "", err
}
