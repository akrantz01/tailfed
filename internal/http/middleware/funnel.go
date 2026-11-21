package middleware

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/netip"

	"tailscale.com/ipn"
)

type connContextKey struct{}

// ConnContext stores the underlying connection in the request context
func ConnContext(ctx context.Context, c net.Conn) context.Context {
	return context.WithValue(ctx, connContextKey{}, c)
}

// FunnelSource replaces the request's remote address with the original client's address when the connection was
// relayed through Tailscale Funnel
func FunnelSource(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if src, ok := funnelSource(r.Context()); ok {
			r.RemoteAddr = src.String()
		}

		next.ServeHTTP(w, r)
	})
}

func funnelSource(ctx context.Context) (netip.AddrPort, bool) {
	conn, ok := ctx.Value(connContextKey{}).(net.Conn)
	if !ok {
		return netip.AddrPort{}, false
	}

	if tlsConn, ok := conn.(*tls.Conn); ok {
		conn = tlsConn.NetConn()
	}

	funnelConn, ok := conn.(*ipn.FunnelConn)
	if !ok {
		return netip.AddrPort{}, false
	}

	return funnelConn.Src, true
}
