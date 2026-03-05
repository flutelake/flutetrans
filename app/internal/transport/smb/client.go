package smb

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path"
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
	if strings.HasPrefix(profile.Path, "/") {
		profile.Path = profile.Path[1:]
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

func (a *Adapter) List(ctx context.Context, client any, listPath string) (models.ListFilesResult, error) {
	c, ok := client.(*conn)
	if !ok || c == nil || c.share == nil {
		return models.ListFilesResult{}, transport.ProtocolError(errors.New("invalid smb client"))
	}
	if strings.TrimSpace(listPath) == "" {
		listPath = "."
	}
	items, err := c.share.ReadDir(listPath)
	if err != nil {
		if ctx.Err() != nil {
			return models.ListFilesResult{}, transport.TimeoutError(ctx.Err())
		}
		return models.ListFilesResult{}, classifySMBError(err)
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
	if !ok || c == nil || c.share == nil {
		return transport.ProtocolError(errors.New("invalid smb client"))
	}
	dirPath = strings.TrimSpace(dirPath)
	if dirPath == "" || dirPath == "." {
		return nil
	}
	clean := path.Clean(dirPath)
	parts := strings.Split(clean, "/")
	current := ""
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if current == "" {
			current = part
		} else {
			current = current + "/" + part
		}
		_ = c.share.Mkdir(current, 0755)
		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}
	}
	return nil
}

func (a *Adapter) Download(ctx context.Context, client any, remotePath string, localPath string, onProgress func(written int64, total int64)) error {
	c, ok := client.(*conn)
	if !ok || c == nil || c.share == nil {
		return transport.ProtocolError(errors.New("invalid smb client"))
	}
	remotePath = strings.TrimSpace(remotePath)
	localPath = strings.TrimSpace(localPath)
	if remotePath == "" || localPath == "" {
		return transport.ValidationError(errors.New("paths required"))
	}

	var total int64
	if st, err := c.share.Stat(remotePath); err == nil {
		total = st.Size()
	}

	src, err := c.share.Open(remotePath)
	if err != nil {
		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}
		return classifySMBError(err)
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
			return classifySMBError(rerr)
		}
	}
	return nil
}

func (a *Adapter) Upload(ctx context.Context, client any, localPath string, remotePath string, onProgress func(written int64, total int64)) error {
	c, ok := client.(*conn)
	if !ok || c == nil || c.share == nil {
		return transport.ProtocolError(errors.New("invalid smb client"))
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

	dst, err := c.share.Create(remotePath)
	if err != nil {
		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}
		return classifySMBError(err)
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
	if !ok || c == nil || c.share == nil {
		return transport.ProtocolError(errors.New("invalid smb client"))
	}
	remotePath = strings.TrimSpace(remotePath)
	if remotePath == "" {
		return transport.ValidationError(errors.New("remotePath required"))
	}
	if recursive {
		if err := c.share.RemoveAll(remotePath); err != nil {
			if ctx.Err() != nil {
				return transport.TimeoutError(ctx.Err())
			}
			return classifySMBError(err)
		}
		return nil
	}
	if err := c.share.Remove(remotePath); err != nil {
		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}
		return classifySMBError(err)
	}
	return nil
}
