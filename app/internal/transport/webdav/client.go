package webdav

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"app/internal/models"
	"app/internal/transport"

	"github.com/studio-b12/gowebdav"
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
	uri := strings.TrimSpace(profile.Host)
	if uri == "" {
		return nil, transport.ValidationError(errors.New("url required"))
	}
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, transport.ValidationError(err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, transport.ValidationError(errors.New("url must be http or https"))
	}

	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return nil, transport.ValidationError(errors.New("url host required"))
	}
	portStr := strings.TrimSpace(parsed.Port())
	port := 0
	if portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, transport.ValidationError(errors.New("invalid url port"))
		}
		port = p
	} else if parsed.Scheme == "https" {
		port = 443
	} else {
		port = 80
	}

	if err := transport.CheckHostReachable(ctx, host, 10*time.Second); err != nil {
		return nil, err
	}
	preflightConn, err := transport.DialTCP(ctx, host, port, 10*time.Second)
	if err != nil {
		return nil, err
	}
	_ = preflightConn.Close()

	username := ""
	password := ""
	if profile.Credentials != nil {
		username = profile.Credentials["username"]
		password = profile.Credentials["password"]
	}
	if username == "" {
		return nil, transport.ValidationError(errors.New("username required"))
	}
	if password == "" {
		return nil, transport.ValidationError(errors.New("password required"))
	}

	timeout := transport.EffectiveTimeout(ctx, 10*time.Second)
	client := gowebdav.NewClient(uri, username, password)
	client.SetTimeout(timeout)
	if err := client.Connect(); err != nil {
		if ctx.Err() != nil {
			return nil, transport.TimeoutError(ctx.Err())
		}
		return nil, classifyWebDAVError(err)
	}
	return client, nil
}

func classifyWebDAVError(err error) error {
	if err == nil {
		return transport.ProtocolError(errors.New("webdav failed"))
	}
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return transport.TimeoutError(err)
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "401") || strings.Contains(msg, "403") || strings.Contains(msg, "unauthorized") || strings.Contains(msg, "forbidden") {
		return transport.AuthError(err)
	}
	return transport.ProtocolError(err)
}

func (a *Adapter) Disconnect(_ context.Context, _ any) error {
	return nil
}
