package ftp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/textproto"
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

func ftpCleanPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "."
	}
	clean := path.Clean(p)
	if clean == "" {
		return "."
	}
	return clean
}

func ftpPathCandidates(p string) []string {
	clean := ftpCleanPath(p)
	if clean == "." {
		return []string{"."}
	}
	if clean == "/" {
		return []string{"/", "."}
	}
	if strings.HasPrefix(clean, "/") {
		rel := strings.TrimPrefix(clean, "/")
		if rel == "" || rel == "." || rel == "/" {
			return []string{clean, "."}
		}
		return []string{clean, rel}
	}
	return []string{clean}
}

func isFTPFileUnavailable(err error) bool {
	var tperr *textproto.Error
	if errors.As(err, &tperr) {
		return tperr.Code == ftplib.StatusFileUnavailable
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "550")
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
	var (
		items    []*ftplib.Entry
		err      error
		usedPath string
	)
	candidates := ftpPathCandidates(listPath)
	for i, candidate := range candidates {
		items, err = conn.List(candidate)
		if err == nil {
			usedPath = candidate
			break
		}
		if i+1 < len(candidates) && isFTPFileUnavailable(err) {
			continue
		}
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
		if usedPath != "." {
			p = path.Join(usedPath, name)
		}
		entries = append(entries, models.FileEntry{
			Name:       name,
			Path:       p,
			IsDir:      it.Type == ftplib.EntryTypeFolder,
			Size:       int64(it.Size),
			ModifiedAt: it.Time.UnixMilli(),
		})
	}
	return models.ListFilesResult{Path: usedPath, Entries: entries}, nil
}

func (a *Adapter) MkdirAll(ctx context.Context, client any, dirPath string) error {
	conn, ok := client.(*ftplib.ServerConn)
	if !ok || conn == nil {
		return transport.ProtocolError(errors.New("invalid ftp client"))
	}
	restore := func() {}
	if cwd, err := conn.CurrentDir(); err == nil && strings.TrimSpace(cwd) != "" {
		restore = func() { _ = conn.ChangeDir(cwd) }
	}
	defer restore()

	candidates := ftpPathCandidates(dirPath)
	for i, candidate := range candidates {
		clean := ftpCleanPath(candidate)
		if clean == "." || clean == "/" {
			return nil
		}

		isAbs := strings.HasPrefix(clean, "/")
		if isAbs {
			if err := conn.ChangeDir("/"); err != nil {
				if i+1 < len(candidates) && isFTPFileUnavailable(err) {
					restore()
					continue
				}
				if ctx.Err() != nil {
					return transport.TimeoutError(ctx.Err())
				}
				return transport.ProtocolError(err)
			}
		} else {
			restore()
		}

		parts := strings.Split(strings.TrimPrefix(clean, "/"), "/")
		okPath := true
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" || part == "." {
				continue
			}
			if ctx.Err() != nil {
				return transport.TimeoutError(ctx.Err())
			}
			if err := conn.ChangeDir(part); err == nil {
				continue
			}

			if err := conn.MakeDir(part); err != nil {
				var tperr *textproto.Error
				if errors.As(err, &tperr) {
					msg := strings.ToLower(tperr.Msg)
					if tperr.Code == ftplib.StatusFileUnavailable && (strings.Contains(msg, "exist") || strings.Contains(msg, "already")) {
						goto ensureDir
					}
				} else {
					msg := strings.ToLower(err.Error())
					if strings.Contains(msg, "exist") || strings.Contains(msg, "already") {
						goto ensureDir
					}
				}
				if i+1 < len(candidates) && isFTPFileUnavailable(err) {
					okPath = false
					break
				}
				if ctx.Err() != nil {
					return transport.TimeoutError(ctx.Err())
				}
				return transport.ProtocolError(err)
			}

		ensureDir:
			if err := conn.ChangeDir(part); err != nil {
				if i+1 < len(candidates) && isFTPFileUnavailable(err) {
					okPath = false
					break
				}
				if ctx.Err() != nil {
					return transport.TimeoutError(ctx.Err())
				}
				return transport.ProtocolError(err)
			}
		}

		if okPath {
			restore()
			return nil
		}
		restore()
	}

	if ctx.Err() != nil {
		return transport.TimeoutError(ctx.Err())
	}
	return transport.ProtocolError(errors.New("create directory failed"))
}

func (a *Adapter) Download(ctx context.Context, client any, remotePath string, localPath string, onProgress func(written int64, total int64)) error {
	conn, ok := client.(*ftplib.ServerConn)
	if !ok || conn == nil {
		return transport.ProtocolError(errors.New("invalid ftp client"))
	}
	localPath = strings.TrimSpace(localPath)
	if strings.TrimSpace(remotePath) == "" || localPath == "" {
		return transport.ValidationError(errors.New("paths required"))
	}

	var (
		total int64
		rc    io.ReadCloser
		err   error
	)
	candidates := ftpPathCandidates(remotePath)
	for i, candidate := range candidates {
		if sz, szErr := conn.FileSize(candidate); szErr == nil {
			total = int64(sz)
		}
		rc, err = conn.Retr(candidate)
		if err == nil {
			break
		}
		if i+1 < len(candidates) && isFTPFileUnavailable(err) {
			continue
		}
		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}
		return transport.ProtocolError(err)
	}
	defer func() {
		if rc != nil {
			_ = rc.Close()
		}
	}()

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
	localPath = strings.TrimSpace(localPath)
	if strings.TrimSpace(remotePath) == "" || localPath == "" {
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

	var lastErr error
	candidates := ftpPathCandidates(remotePath)
	for i, candidate := range candidates {
		if err := conn.Stor(candidate, reader); err == nil {
			return nil
		} else {
			lastErr = err
			if i+1 < len(candidates) && isFTPFileUnavailable(err) {
				if _, seekErr := src.Seek(0, 0); seekErr == nil {
					reader = io.Reader(src)
					if onProgress != nil {
						reader = &progressReader{r: src, total: total, cb: onProgress}
					}
				}
				continue
			}
			if ctx.Err() != nil {
				return transport.TimeoutError(ctx.Err())
			}
			return transport.ProtocolError(err)
		}
	}
	if ctx.Err() != nil {
		return transport.TimeoutError(ctx.Err())
	}
	return transport.ProtocolError(lastErr)
}

func (a *Adapter) Remove(ctx context.Context, client any, remotePath string, recursive bool) error {
	conn, ok := client.(*ftplib.ServerConn)
	if !ok || conn == nil {
		return transport.ProtocolError(errors.New("invalid ftp client"))
	}
	if strings.TrimSpace(remotePath) == "" {
		return transport.ValidationError(errors.New("remotePath required"))
	}

	if !recursive {
		var lastErr error
		candidates := ftpPathCandidates(remotePath)
		for i, candidate := range candidates {
			if err := conn.Delete(candidate); err == nil {
				return nil
			} else {
				lastErr = err
			}
			if err := conn.RemoveDir(candidate); err == nil {
				return nil
			} else {
				lastErr = err
			}
			if i+1 < len(candidates) && isFTPFileUnavailable(lastErr) {
				continue
			}
			if ctx.Err() != nil {
				return transport.TimeoutError(ctx.Err())
			}
			return transport.ProtocolError(errors.New("delete failed"))
		}
		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}
		return transport.ProtocolError(lastErr)
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

	var lastErr error
	candidates := ftpPathCandidates(remotePath)
	for i, candidate := range candidates {
		if err := conn.Delete(candidate); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if err := conn.RemoveDir(candidate); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if err := removeDirRecursive(candidate); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if i+1 < len(candidates) && isFTPFileUnavailable(lastErr) {
			continue
		}
		return lastErr
	}
	return lastErr
}
