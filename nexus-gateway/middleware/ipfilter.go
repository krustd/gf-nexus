package middleware

import (
	"net"

	"github.com/gogf/gf/v2/net/ghttp"

	"github.com/krustd/gf-nexus/nexus-gateway/config"
	"github.com/krustd/gf-nexus/nexus-gateway/internal"
)

type ipEntry struct {
	ip   net.IP     // 精确 IP
	cidr *net.IPNet // CIDR 网段
}

// IPFilter IP 黑白名单过滤（动态读取配置）
func IPFilter(holder *config.DynamicConfigHolder) ghttp.HandlerFunc {
	// 缓存已解析的 IP 条目，避免每次请求重复解析
	var (
		cachedCfg     *config.IPFilterConfig
		cachedEntries []ipEntry
	)

	return func(r *ghttp.Request) {
		cfg := holder.Load().IPFilter

		if !cfg.Enabled {
			r.Middleware.Next()
			return
		}

		// 配置变更时重新解析
		if cachedCfg == nil || !ipFilterEqual(cachedCfg, &cfg) {
			cachedEntries = parseIPEntries(cfg.Addresses)
			cfgCopy := cfg
			cachedCfg = &cfgCopy
		}

		clientIP := net.ParseIP(r.GetClientIp())
		if clientIP == nil {
			r.Middleware.Next()
			return
		}

		matched := matchIP(clientIP, cachedEntries)

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

func ipFilterEqual(a *config.IPFilterConfig, b *config.IPFilterConfig) bool {
	if a.Enabled != b.Enabled || a.Mode != b.Mode {
		return false
	}
	if len(a.Addresses) != len(b.Addresses) {
		return false
	}
	for i := range a.Addresses {
		if a.Addresses[i] != b.Addresses[i] {
			return false
		}
	}
	return true
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
