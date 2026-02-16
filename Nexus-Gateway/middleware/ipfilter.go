package middleware

import (
	"net"

	"github.com/gogf/gf/v2/net/ghttp"

	"github.com/krustd/nexus-gateway/config"
	"github.com/krustd/nexus-gateway/internal"
)

type ipEntry struct {
	ip   net.IP   // 精确 IP
	cidr *net.IPNet // CIDR 网段
}

// IPFilter IP 黑白名单过滤
func IPFilter(cfg config.IPFilterConfig) ghttp.HandlerFunc {
	if !cfg.Enabled {
		return passthrough
	}

	entries := parseIPEntries(cfg.Addresses)

	return func(r *ghttp.Request) {
		clientIP := net.ParseIP(r.GetClientIp())
		if clientIP == nil {
			r.Middleware.Next()
			return
		}

		matched := matchIP(clientIP, entries)

		switch cfg.Mode {
		case "whitelist":
			if !matched {
				internal.GatewayError(r, internal.CodeIPBlocked, "ip not allowed")
				return
			}
		case "blacklist":
			if matched {
				internal.GatewayError(r, internal.CodeIPBlocked, "ip blocked")
				return
			}
		}

		r.Middleware.Next()
	}
}

func parseIPEntries(addresses []string) []ipEntry {
	entries := make([]ipEntry, 0, len(addresses))
	for _, addr := range addresses {
		_, cidr, err := net.ParseCIDR(addr)
		if err == nil {
			entries = append(entries, ipEntry{cidr: cidr})
			continue
		}
		ip := net.ParseIP(addr)
		if ip != nil {
			entries = append(entries, ipEntry{ip: ip})
		}
	}
	return entries
}

func matchIP(clientIP net.IP, entries []ipEntry) bool {
	for _, e := range entries {
		if e.cidr != nil && e.cidr.Contains(clientIP) {
			return true
		}
		if e.ip != nil && e.ip.Equal(clientIP) {
			return true
		}
	}
	return false
}
