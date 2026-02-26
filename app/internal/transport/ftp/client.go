package ftp

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

func (a *Adapter) List(ctx context.Context, client any, listPath string) (models.ListFilesResult, error) {
	conn, ok := client.(*ftplib.ServerConn)
	if !ok || conn == nil {
		return models.ListFilesResult{}, transport.ProtocolError(errors.New("invalid ftp client"))
	}
	if strings.TrimSpace(listPath) == "" {
		listPath = "."
	}
	items, err := conn.List(listPath)
	if err != nil {
		if ctx.Err() != nil {
			return models.ListFilesResult{}, transport.TimeoutError(ctx.Err())
		}
		return models.ListFilesResult{}, transport.ProtocolError(err)
	}

	entries := make([]models.FileEntry, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}
		name := it.Name
		p := name
		if listPath != "." {
			p = path.Join(listPath, name)
		}
		entries = append(entries, models.FileEntry{
			Name:       name,
			Path:       p,
			IsDir:      it.Type == ftplib.EntryTypeFolder,
			Size:       int64(it.Size),
			ModifiedAt: it.Time.UnixMilli(),
		})
	}
	return models.ListFilesResult{Path: listPath, Entries: entries}, nil
}

func (a *Adapter) MkdirAll(ctx context.Context, client any, dirPath string) error {
	conn, ok := client.(*ftplib.ServerConn)
	if !ok || conn == nil {
		return transport.ProtocolError(errors.New("invalid ftp client"))
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
		_ = conn.MakeDir(current)
		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}
	}
	return nil
}

func (a *Adapter) Download(ctx context.Context, client any, remotePath string, localPath string, onProgress func(written int64, total int64)) error {
	conn, ok := client.(*ftplib.ServerConn)
	if !ok || conn == nil {
		return transport.ProtocolError(errors.New("invalid ftp client"))
	}
	remotePath = strings.TrimSpace(remotePath)
	localPath = strings.TrimSpace(localPath)
	if remotePath == "" || localPath == "" {
		return transport.ValidationError(errors.New("paths required"))
	}

	var total int64
	if sz, err := conn.FileSize(remotePath); err == nil {
		total = int64(sz)
	}

	rc, err := conn.Retr(remotePath)
	if err != nil {
		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}
		return transport.ProtocolError(err)
	}
	defer rc.Close()

	dst, err := os.Create(localPath)
	if err != nil {
		return transport.ProtocolError(err)
	}
	defer dst.Close()

	buf := make([]byte, 256*1024)
	var written int64
	for {
		n, rerr := rc.Read(buf)
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

type progressReader struct {
	r     io.Reader
	total int64
	cb    func(written int64, total int64)
	n     int64
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.n += int64(n)
		if p.cb != nil {
			p.cb(p.n, p.total)
		}
	}
	return n, err
}

func (a *Adapter) Upload(ctx context.Context, client any, localPath string, remotePath string, onProgress func(written int64, total int64)) error {
	conn, ok := client.(*ftplib.ServerConn)
	if !ok || conn == nil {
		return transport.ProtocolError(errors.New("invalid ftp client"))
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

	reader := io.Reader(src)
	if onProgress != nil {
		reader = &progressReader{r: src, total: total, cb: onProgress}
	}

	if err := conn.Stor(remotePath, reader); err != nil {
		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}
		return transport.ProtocolError(err)
	}
	return nil
}

func (a *Adapter) Remove(ctx context.Context, client any, remotePath string, recursive bool) error {
	conn, ok := client.(*ftplib.ServerConn)
	if !ok || conn == nil {
		return transport.ProtocolError(errors.New("invalid ftp client"))
	}
	remotePath = strings.TrimSpace(remotePath)
	if remotePath == "" {
		return transport.ValidationError(errors.New("remotePath required"))
	}

	if !recursive {
		if err := conn.Delete(remotePath); err == nil {
			return nil
		}
		if err := conn.RemoveDir(remotePath); err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}
		return transport.ProtocolError(errors.New("delete failed"))
	}

	var removeDirRecursive func(p string) error
	removeDirRecursive = func(p string) error {
		items, err := conn.List(p)
		if err != nil {
			if ctx.Err() != nil {
				return transport.TimeoutError(ctx.Err())
			}
			return transport.ProtocolError(err)
		}
		for _, it := range items {
			if it == nil {
				continue
			}
			name := strings.TrimSpace(it.Name)
			if name == "" || name == "." || name == ".." {
				continue
			}
			child := path.Join(p, name)
			if it.Type == ftplib.EntryTypeFolder {
				if err := removeDirRecursive(child); err != nil {
					return err
				}
				continue
			}
			if err := conn.Delete(child); err != nil {
				if ctx.Err() != nil {
					return transport.TimeoutError(ctx.Err())
				}
				return transport.ProtocolError(err)
			}
			if ctx.Err() != nil {
				return transport.TimeoutError(ctx.Err())
			}
		}
		if err := conn.RemoveDir(p); err != nil {
			if ctx.Err() != nil {
				return transport.TimeoutError(ctx.Err())
			}
			return transport.ProtocolError(err)
		}
		return nil
	}

	if err := conn.Delete(remotePath); err == nil {
		return nil
	}
	if err := conn.RemoveDir(remotePath); err == nil {
		return nil
	}
	return removeDirRecursive(remotePath)
}
