package utils

import (
	"net"
	"net/http"
	"time"
)

// CreateOptimizedTransport returns an optimized HTTP transport for better performance
func CreateOptimizedTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,  // Dialing timeout
			KeepAlive: 30 * time.Second, // Connection keep-alive time
			DualStack: true,             // Support IPv4 and IPv6
		}).DialContext,
		MaxIdleConns:          100,              // Maximum idle connections
		IdleConnTimeout:       90 * time.Second, // Idle connection timeout
		TLSHandshakeTimeout:   5 * time.Second,  // TLS handshake timeout
		ExpectContinueTimeout: 1 * time.Second,  // 100-continue timeout
		MaxIdleConnsPerHost:   10,               // Maximum idle connections per host
		DisableKeepAlives:     false,            // Enable connection reuse
	}
}

// CreateHTTPClient creates an HTTP client with optimized settings
func CreateHTTPClient() *http.Client {
	return &http.Client{
		Transport: CreateOptimizedTransport(),
		Timeout:   time.Second * 30,
	}
}
