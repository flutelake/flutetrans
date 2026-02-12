package nfs

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"app/internal/models"
	"app/internal/transport"

	"github.com/vmware/go-nfs-client/nfs"
	"github.com/vmware/go-nfs-client/nfs/rpc"
)

type Adapter struct{}

type conn struct {
	mount  *nfs.Mount
	target *nfs.Target
}

func NewAdapter() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Test(ctx context.Context, profile models.ConnectionProfile) (time.Duration, error) {
	started := time.Now()
	client, err := a.Connect(ctx, profile)
	if err != nil {
		return 0, err
	}
	_ = a.Disconnect(ctx, client)
	return time.Since(started), nil
}

func (a *Adapter) Connect(ctx context.Context, profile models.ConnectionProfile) (any, error) {
	host := strings.TrimSpace(profile.Host)
	if host == "" {
		return nil, transport.ValidationError(errors.New("host required"))
	}

	if err := transport.CheckHostReachable(ctx, host, 10*time.Second); err != nil {
		return nil, err
	}
	port := profile.Port
	if port == 0 {
		port = 111
	}
	preflightConn, err := transport.DialTCP(ctx, host, port, 10*time.Second)
	if err != nil {
		return nil, err
	}
	_ = preflightConn.Close()

	exportPath := strings.TrimSpace(profile.Path)
	if exportPath == "" {
		return nil, transport.ValidationError(errors.New("exportPath required"))
	}

	addr := host
	if profile.Port != 0 {
		addr = net.JoinHostPort(host, fmt.Sprintf("%d", profile.Port))
	}

	timeout := transport.EffectiveTimeout(ctx, 10*time.Second)

	type result struct {
		mount  *nfs.Mount
		target *nfs.Target
		err    error
	}
	ch := make(chan result, 1)
	go func() {
		m, err := nfs.DialMount(addr)
		if err != nil {
			ch <- result{err: err}
			return
		}
		m.SetTimeout(timeout)
		t, err := m.Mount(exportPath, rpc.AuthNull)
		if err != nil {
			_ = m.Close()
			ch <- result{err: err}
			return
		}
		_, err = t.FSInfo()
		if err != nil {
			_ = t.Close()
			_ = m.Unmount()
			_ = m.Close()
			ch <- result{err: err}
			return
		}
		ch <- result{mount: m, target: t}
	}()

	select {
	case <-ctx.Done():
		return nil, transport.TimeoutError(ctx.Err())
	case r := <-ch:
		if r.err != nil {
			if errors.Is(r.err, context.DeadlineExceeded) || errors.Is(r.err, context.Canceled) {
				return nil, transport.TimeoutError(r.err)
			}
			return nil, transport.ProtocolError(r.err)
		}
		return &conn{mount: r.mount, target: r.target}, nil
	}
}

func (a *Adapter) Disconnect(_ context.Context, client any) error {
	c, ok := client.(*conn)
	if !ok || c == nil {
		return nil
	}
	if c.target != nil {
		_ = c.target.Close()
	}
	if c.mount != nil {
		_ = c.mount.Unmount()
		return c.mount.Close()
	}
	return nil
}
