package sftp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
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

func (a *Adapter) List(ctx context.Context, client any, listPath string) (models.ListFilesResult, error) {
	c, ok := client.(*conn)
	if !ok || c == nil || c.sftp == nil {
		return models.ListFilesResult{}, transport.ProtocolError(errors.New("invalid sftp client"))
	}
	if strings.TrimSpace(listPath) == "" {
		listPath = "."
	}
	items, err := c.sftp.ReadDir(listPath)
	if err != nil {
		if ctx.Err() != nil {
			return models.ListFilesResult{}, transport.TimeoutError(ctx.Err())
		}
		return models.ListFilesResult{}, transport.ProtocolError(err)
	}

	entries := make([]models.FileEntry, 0, len(items))
	for _, fi := range items {
		p := fi.Name()
		if listPath != "." {
			p = path.Join(listPath, fi.Name())
		}
		entries = append(entries, models.FileEntry{
			Name:       fi.Name(),
			Path:       p,
			IsDir:      fi.IsDir(),
			Size:       fi.Size(),
			ModifiedAt: fi.ModTime().UnixMilli(),
		})
	}

	return models.ListFilesResult{Path: listPath, Entries: entries}, nil
}

func (a *Adapter) MkdirAll(ctx context.Context, client any, dirPath string) error {
	c, ok := client.(*conn)
	if !ok || c == nil || c.sftp == nil {
		return transport.ProtocolError(errors.New("invalid sftp client"))
	}
	if strings.TrimSpace(dirPath) == "" || dirPath == "." {
		return nil
	}
	if err := c.sftp.MkdirAll(dirPath); err != nil {
		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}
		return transport.ProtocolError(err)
	}
	return nil
}

func (a *Adapter) Download(ctx context.Context, client any, remotePath string, localPath string, onProgress func(written int64, total int64)) error {
	c, ok := client.(*conn)
	if !ok || c == nil || c.sftp == nil {
		return transport.ProtocolError(errors.New("invalid sftp client"))
	}
	remotePath = strings.TrimSpace(remotePath)
	localPath = strings.TrimSpace(localPath)
	if remotePath == "" || localPath == "" {
		return transport.ValidationError(errors.New("paths required"))
	}

	var total int64
	if st, err := c.sftp.Stat(remotePath); err == nil {
		total = st.Size()
	}

	src, err := c.sftp.Open(remotePath)
	if err != nil {
		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}
		return transport.ProtocolError(err)
	}
	defer src.Close()

	dst, err := os.Create(localPath)
	if err != nil {
		return transport.ProtocolError(err)
	}
	defer dst.Close()

	buf := make([]byte, 256*1024)
	var written int64
	for {
		n, rerr := src.Read(buf)
		if n > 0 {
			wn, werr := dst.Write(buf[:n])
			if werr != nil {
				return transport.ProtocolError(werr)
			}
			written += int64(wn)
			if onProgress != nil {
				onProgress(written, total)
			}
		}
		if rerr != nil {
			if errors.Is(rerr, io.EOF) {
				break
			}
			if ctx.Err() != nil {
				return transport.TimeoutError(ctx.Err())
			}
			return transport.ProtocolError(rerr)
		}
	}
	return nil
}

func (a *Adapter) Upload(ctx context.Context, client any, localPath string, remotePath string, onProgress func(written int64, total int64)) error {
	c, ok := client.(*conn)
	if !ok || c == nil || c.sftp == nil {
		return transport.ProtocolError(errors.New("invalid sftp client"))
	}
	remotePath = strings.TrimSpace(remotePath)
	localPath = strings.TrimSpace(localPath)
	if remotePath == "" || localPath == "" {
		return transport.ValidationError(errors.New("paths required"))
	}

	src, err := os.Open(localPath)
	if err != nil {
		return transport.ProtocolError(err)
	}
	defer src.Close()

	var total int64
	if st, err := src.Stat(); err == nil {
		total = st.Size()
	}

	dst, err := c.sftp.Create(remotePath)
	if err != nil {
		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}
		return transport.ProtocolError(err)
	}
	defer dst.Close()

	buf := make([]byte, 256*1024)
	var written int64
	for {
		n, rerr := src.Read(buf)
		if n > 0 {
			wn, werr := dst.Write(buf[:n])
			if werr != nil {
				return transport.ProtocolError(werr)
			}
			written += int64(wn)
			if onProgress != nil {
				onProgress(written, total)
			}
		}
		if rerr != nil {
			if errors.Is(rerr, io.EOF) {
				break
			}
			if ctx.Err() != nil {
				return transport.TimeoutError(ctx.Err())
			}
			return transport.ProtocolError(rerr)
		}
	}
	return nil
}

func (a *Adapter) Remove(ctx context.Context, client any, remotePath string, recursive bool) error {
	c, ok := client.(*conn)
	if !ok || c == nil || c.sftp == nil {
		return transport.ProtocolError(errors.New("invalid sftp client"))
	}
	remotePath = strings.TrimSpace(remotePath)
	if remotePath == "" {
		return transport.ValidationError(errors.New("remotePath required"))
	}

	if !recursive {
		if err := c.sftp.Remove(remotePath); err == nil {
			return nil
		}
		if err := c.sftp.RemoveDirectory(remotePath); err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}
		return transport.ProtocolError(errors.New("delete failed"))
	}

	var removeDirRecursive func(p string) error
	removeDirRecursive = func(p string) error {
		items, err := c.sftp.ReadDir(p)
		if err != nil {
			if ctx.Err() != nil {
				return transport.TimeoutError(ctx.Err())
			}
			if rerr := c.sftp.Remove(p); rerr == nil {
				return nil
			}
			return transport.ProtocolError(err)
		}
		for _, fi := range items {
			if fi == nil {
				continue
			}
			name := strings.TrimSpace(fi.Name())
			if name == "" || name == "." || name == ".." {
				continue
			}
			child := path.Join(p, name)
			if fi.IsDir() {
				if err := removeDirRecursive(child); err != nil {
					return err
				}
				continue
			}
			if err := c.sftp.Remove(child); err != nil {
				if ctx.Err() != nil {
					return transport.TimeoutError(ctx.Err())
				}
				return transport.ProtocolError(err)
			}
			if ctx.Err() != nil {
				return transport.TimeoutError(ctx.Err())
			}
		}
		if err := c.sftp.RemoveDirectory(p); err != nil {
			if ctx.Err() != nil {
				return transport.TimeoutError(ctx.Err())
			}
			return transport.ProtocolError(err)
		}
		return nil
	}

	if err := c.sftp.Remove(remotePath); err == nil {
		return nil
	}
	if err := c.sftp.RemoveDirectory(remotePath); err == nil {
		return nil
	}
	return removeDirRecursive(remotePath)
}
