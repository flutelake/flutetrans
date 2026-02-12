package ftp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"app/internal/models"
	"app/internal/transport"

	ftplib "github.com/jlaffaye/ftp"
)

type Adapter struct{}

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
	host := profile.Host
	if host == "" {
		return nil, transport.ValidationError(errors.New("host required"))
	}
	port := profile.Port
	if port == 0 {
		port = 21
	}

	if err := transport.CheckHostReachable(ctx, host, 10*time.Second); err != nil {
		return nil, err
	}
	preflightConn, err := transport.DialTCP(ctx, host, port, 10*time.Second)
	if err != nil {
		return nil, err
	}
	_ = preflightConn.Close()

	timeout := transport.EffectiveTimeout(ctx, 10*time.Second)
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	conn, err := ftplib.Dial(addr, ftplib.DialWithTimeout(timeout))
	if err != nil {
		if ctx.Err() != nil {
			return nil, transport.TimeoutError(ctx.Err())
		}
		return nil, transport.ProtocolError(err)
	}

	username := ""
	password := ""
	if profile.Credentials != nil {
		username = profile.Credentials["username"]
		password = profile.Credentials["password"]
	}
	if username == "" {
		_ = conn.Quit()
		return nil, transport.ValidationError(errors.New("username required"))
	}
	if password == "" {
		_ = conn.Quit()
		return nil, transport.ValidationError(errors.New("password required"))
	}

	if err := conn.Login(username, password); err != nil {
		_ = conn.Quit()
		return nil, transport.AuthError(err)
	}
	return conn, nil
}

func (a *Adapter) Disconnect(_ context.Context, client any) error {
	conn, ok := client.(*ftplib.ServerConn)
	if !ok || conn == nil {
		return nil
	}
	return conn.Quit()
}
