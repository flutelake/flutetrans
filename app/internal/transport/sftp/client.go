package sftp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"app/internal/models"
	"app/internal/transport"

	sftplib "github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type Adapter struct{}

type conn struct {
	ssh  *ssh.Client
	sftp *sftplib.Client
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
	host := profile.Host
	if host == "" {
		return nil, transport.ValidationError(errors.New("host required"))
	}
	port := profile.Port
	if port == 0 {
		port = 22
	}

	if err := transport.CheckHostReachable(ctx, host, 10*time.Second); err != nil {
		return nil, err
	}

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	tcpConn, err := transport.DialTCP(ctx, host, port, 10*time.Second)
	if err != nil {
		return nil, err
	}

	username := ""
	password := ""
	privateKeyPath := ""
	passphrase := ""
	if profile.Credentials != nil {
		username = profile.Credentials["username"]
		password = profile.Credentials["password"]
		privateKeyPath = profile.Credentials["privateKeyPath"]
		passphrase = profile.Credentials["passphrase"]
	}
	if username == "" {
		_ = tcpConn.Close()
		return nil, transport.ValidationError(errors.New("username required"))
	}

	var authMethods []ssh.AuthMethod
	if privateKeyPath != "" {
		keyData, err := os.ReadFile(privateKeyPath)
		if err != nil {
			_ = tcpConn.Close()
			return nil, transport.ValidationError(fmt.Errorf("failed to read private key: %w", err))
		}
		var signer ssh.Signer
		if passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(keyData)
		}
		if err != nil {
			_ = tcpConn.Close()
			return nil, transport.AuthError(err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	} else {
		if password == "" {
			_ = tcpConn.Close()
			return nil, transport.ValidationError(errors.New("password required"))
		}
		authMethods = append(authMethods, ssh.Password(password))
	}

	timeout := transport.EffectiveTimeout(ctx, 10*time.Second)
	config := &ssh.ClientConfig{
		User:            username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	_ = tcpConn.SetDeadline(time.Now().Add(timeout))
	sshConn, chans, reqs, err := ssh.NewClientConn(tcpConn, addr, config)
	if err != nil {
		_ = tcpConn.Close()
		return nil, classifyHandshakeError(err)
	}
	_ = tcpConn.SetDeadline(time.Time{})
	sshClient := ssh.NewClient(sshConn, chans, reqs)

	sftpClient, err := sftplib.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, transport.ProtocolError(err)
	}
	return &conn{ssh: sshClient, sftp: sftpClient}, nil
}

func classifyHandshakeError(err error) error {
	if err == nil {
		return transport.ProtocolError(errors.New("handshake failed"))
	}
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return transport.TimeoutError(err)
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unable to authenticate") || strings.Contains(msg, "no supported methods remain") || strings.Contains(msg, "permission denied") {
		return transport.AuthError(err)
	}
	return transport.ProtocolError(err)
}

func (a *Adapter) Disconnect(_ context.Context, client any) error {
	c, ok := client.(*conn)
	if !ok || c == nil {
		return nil
	}
	if c.sftp != nil {
		_ = c.sftp.Close()
	}
	if c.ssh != nil {
		return c.ssh.Close()
	}
	return nil
}
