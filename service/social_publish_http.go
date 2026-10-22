package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"time"
)

const socialRequestTimeout = 30 * time.Second

var socialPublishTransport = &http.Transport{
	// No environment proxy: DNS validation must describe the actual peer.
	DialContext:           dialSocialPublic,
	TLSHandshakeTimeout:   10 * time.Second,
	ResponseHeaderTimeout: socialRequestTimeout,
	IdleConnTimeout:       90 * time.Second,
	MaxIdleConns:          20,
}

func socialPublishClient(client *http.Client) *http.Client {
	result := http.Client{Transport: socialPublishTransport, Timeout: socialRequestTimeout}
	if client != nil {
		result = *client
		if result.Transport == nil {
			result.Transport = socialPublishTransport
		}
		if result.Timeout <= 0 || result.Timeout > socialRequestTimeout {
			result.Timeout = socialRequestTimeout
		}
	}
	result.Jar = nil
	result.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &result
}

func socialPublicIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.Zone() != "" {
		return false
	}
	// Reject special-use, documentation, benchmarking and transition ranges too.
	for _, raw := range []string{"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "192.88.99.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "2001::/23", "2001:db8::/32", "2002::/16", "3fff::/20"} {
		if netip.MustParsePrefix(raw).Contains(ip) {
			return false
		}
	}
	return ip.Is4() || netip.MustParsePrefix("2000::/3").Contains(ip)
}

func dialSocialPublic(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid publishing host")
	}
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil || len(ips) == 0 {
		return nil, fmt.Errorf("publishing host lookup failed")
	}
	dialer := net.Dialer{Timeout: 10 * time.Second}
	return dialSocialAddresses(ctx, network, port, ips, dialer.DialContext)
}

func dialSocialAddresses(ctx context.Context, network, port string, ips []netip.Addr, dial func(context.Context, string, string) (net.Conn, error)) (net.Conn, error) {
	for _, ip := range ips {
		if !socialPublicIP(ip) {
			return nil, fmt.Errorf("publishing host must resolve to public IP addresses")
		}
	}
	// Dial the checked numeric address, not the hostname: no second DNS lookup.
	// The transport retains the original Host and TLS ServerName verification.
	for _, ip := range ips {
		conn, err := dial(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
	}
	return nil, fmt.Errorf("publishing connection failed")
}
