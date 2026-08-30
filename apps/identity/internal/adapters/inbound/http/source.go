package httpadapter

import (
	"net"
	"net/http"
	"strings"
)

// requestSource returns a coarse source prefix. Forwarded headers are used
// only when the deployment explicitly marks its ingress as trusted.
func requestSource(request *http.Request, trustProxyHeaders bool) string {
	address := request.RemoteAddr
	if trustProxyHeaders {
		if forwarded := strings.TrimSpace(strings.Split(request.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
			address = forwarded
		}
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		host = strings.TrimSpace(address)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return "unknown"
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		return (&net.IPNet{IP: ipv4.Mask(net.CIDRMask(24, 32)), Mask: net.CIDRMask(24, 32)}).String()
	}
	return (&net.IPNet{IP: ip.Mask(net.CIDRMask(56, 128)), Mask: net.CIDRMask(56, 128)}).String()
}
