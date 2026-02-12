package transport

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

func normalizeHost(host string) string {
	h := strings.TrimSpace(host)
	h = strings.TrimPrefix(h, "[")
	h = strings.TrimSuffix(h, "]")
	return h
}

func contextOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func CheckHostReachable(ctx context.Context, host string, fallback time.Duration) error {
	h := normalizeHost(host)
	timeout := EffectiveTimeout(ctx, fallback)

	rctx, cancel := context.WithTimeout(contextOrBackground(ctx), timeout)
	defer cancel()

	_, err := net.DefaultResolver.LookupIPAddr(rctx, h)
	if err != nil {
		if rctx.Err() != nil {
			return TimeoutError(rctx.Err())
		}
		return ProtocolError(fmt.Errorf("host unreachable: %w", err))
	}
	return nil
}

func DialTCP(ctx context.Context, host string, port int, fallback time.Duration) (net.Conn, error) {
	h := normalizeHost(host)
	timeout := EffectiveTimeout(ctx, fallback)

	dctx, cancel := context.WithTimeout(contextOrBackground(ctx), timeout)
	defer cancel()

	addr := net.JoinHostPort(h, strconv.Itoa(port))
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(dctx, "tcp", addr)
	if err != nil {
		if dctx.Err() != nil {
			return nil, TimeoutError(dctx.Err())
		}
		return nil, ProtocolError(fmt.Errorf("port unreachable: %w", err))
	}
	return conn, nil
}
