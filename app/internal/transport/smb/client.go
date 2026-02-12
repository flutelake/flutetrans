package smb

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"app/internal/models"
	"app/internal/transport"

	"github.com/hirochachacha/go-smb2"
)

type Adapter struct{}

type conn struct {
	netConn net.Conn
	session *smb2.Session
	share   *smb2.Share
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
	port := profile.Port
	if port == 0 {
		port = 445
	}

	if err := transport.CheckHostReachable(ctx, host, 10*time.Second); err != nil {
		return nil, err
	}

	netConn, err := transport.DialTCP(ctx, host, port, 10*time.Second)
	if err != nil {
		return nil, err
	}

	username := ""
	password := ""
	domain := ""
	if profile.Credentials != nil {
		username = profile.Credentials["username"]
		password = profile.Credentials["password"]
		domain = profile.Credentials["domain"]
	}
	if username == "" {
		_ = netConn.Close()
		return nil, transport.ValidationError(errors.New("username required"))
	}
	if password == "" {
		_ = netConn.Close()
		return nil, transport.ValidationError(errors.New("password required"))
	}

	shareName := strings.TrimSpace(profile.Path)
	if shareName == "" {
		_ = netConn.Close()
		return nil, transport.ValidationError(errors.New("share required"))
	}

	dialer := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     username,
			Password: password,
			Domain:   domain,
		},
	}
	serverSession, err := dialer.Dial(netConn)
	if err != nil {
		_ = netConn.Close()
		return nil, classifySMBError(err)
	}

	share, err := serverSession.Mount(shareName)
	if err != nil {
		_ = serverSession.Logoff()
		_ = netConn.Close()
		return nil, classifySMBError(err)
	}

	_, _ = share.ReadDir(".")
	return &conn{netConn: netConn, session: serverSession, share: share}, nil
}

func classifySMBError(err error) error {
	if err == nil {
		return transport.ProtocolError(errors.New("smb failed"))
	}
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return transport.TimeoutError(err)
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "logon") || strings.Contains(msg, "access denied") || strings.Contains(msg, "status_logon_failure") {
		return transport.AuthError(err)
	}
	return transport.ProtocolError(err)
}

func (a *Adapter) Disconnect(_ context.Context, client any) error {
	c, ok := client.(*conn)
	if !ok || c == nil {
		return nil
	}
	if c.share != nil {
		_ = c.share.Umount()
	}
	if c.session != nil {
		_ = c.session.Logoff()
	}
	if c.netConn != nil {
		return c.netConn.Close()
	}
	return nil
}
