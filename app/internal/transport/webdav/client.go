package webdav

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
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

func webdavProbeCandidates(connectPath string, username string, allowFallback bool) []string {
	connectPath = strings.TrimSpace(connectPath)
	if connectPath == "" {
		connectPath = "/"
	}
	if !strings.HasPrefix(connectPath, "/") {
		connectPath = "/" + connectPath
	}
	connectPath = path.Clean(connectPath)
	if connectPath == "." {
		connectPath = "/"
	}
	candidates := []string{connectPath}
	if connectPath != "/" {
		candidates = append(candidates, connectPath+"/")
	}
	if !allowFallback {
		return candidates
	}
	if connectPath != "/" {
		return candidates
	}

	common := []string{
		"/webdav",
		"/webdav/",
		"/dav",
		"/dav/",
		"/remote.php/webdav",
		"/remote.php/webdav/",
		"/remote.php/dav",
		"/remote.php/dav/",
	}
	if strings.TrimSpace(username) != "" {
		u := url.PathEscape(username)
		common = append(common,
			"/remote.php/dav/files/"+u,
			"/remote.php/dav/files/"+u+"/",
		)
	}

	seen := map[string]struct{}{}
	out := make([]string, 0, len(candidates)+len(common))
	for _, p := range append(candidates, common...) {
		p = path.Clean(strings.TrimSpace(p))
		if p == "." {
			p = "/"
		}
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func webdavOptionsProbe(ctx context.Context, base *url.URL, username string, password string, probePath string, timeout time.Duration) (bool, error) {
	if base == nil {
		return false, errors.New("invalid url")
	}
	if strings.TrimSpace(probePath) == "" {
		probePath = "/"
	}
	u := *base
	u.Path = probePath
	u.RawQuery = ""
	u.Fragment = ""

	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	httpClient := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("stopped after too many redirects")
			}
			req.Header.Set("Authorization", "Basic "+auth)
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodOptions, u.String(), nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := httpClient.Do(req)
	if err != nil {
		return false, err
	}
	_ = resp.Body.Close()

	dav := strings.TrimSpace(resp.Header.Get("DAV"))
	allow := strings.ToUpper(resp.Header.Get("Allow"))
	if dav != "" || strings.Contains(allow, "PROPFIND") || strings.Contains(allow, "MKCOL") {
		return true, nil
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return false, transport.AuthError(fmt.Errorf("webdav auth failed: %s", resp.Status))
	}
	return false, fmt.Errorf("webdav options probe failed: %s", resp.Status)
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
	parsed, parseErr := url.Parse(uri)
	if parseErr != nil {
		return nil, transport.ValidationError(parseErr)
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
		p, parseErr := strconv.Atoi(portStr)
		if parseErr != nil {
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
	connectPath := strings.TrimSpace(profile.Path)
	if connectPath == "" {
		connectPath = strings.TrimSpace(parsed.Path)
	}
	base := &url.URL{
		Scheme: parsed.Scheme,
		Host:   parsed.Host,
	}
	allowFallback := strings.TrimSpace(profile.Path) == "" && strings.TrimSpace(parsed.Path) == ""
	candidates := webdavProbeCandidates(connectPath, username, allowFallback)

	var probeErr error
	for _, p := range candidates {
		if _, statErr := client.Stat(p); statErr == nil {
			probeErr = nil
			break
		} else {
			probeErr = statErr
			msg := strings.ToLower(statErr.Error())
			if strings.Contains(msg, "401") || strings.Contains(msg, "403") || strings.Contains(msg, "unauthorized") || strings.Contains(msg, "forbidden") {
				return nil, transport.AuthError(statErr)
			}
			if strings.Contains(msg, "405") || strings.Contains(msg, "method not allowed") {
				if ok, optErr := webdavOptionsProbe(ctx, base, username, password, p, timeout); ok && optErr == nil {
					probeErr = nil
					break
				} else if optErr != nil {
					probeErr = optErr
				}
				continue
			}
		}
	}
	if probeErr != nil {
		if ctx.Err() != nil {
			return nil, transport.TimeoutError(ctx.Err())
		}
		return nil, classifyWebDAVError(probeErr)
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
	if strings.Contains(msg, "405") || strings.Contains(msg, "method not allowed") {
		return transport.ValidationError(fmt.Errorf("webdav not enabled at url/path (405): %w", err))
	}
	if strings.Contains(msg, "404") || strings.Contains(msg, "not found") {
		return transport.ValidationError(fmt.Errorf("webdav endpoint not found (404): %w", err))
	}
	return transport.ProtocolError(err)
}

func (a *Adapter) Disconnect(_ context.Context, _ any) error {
	return nil
}

func (a *Adapter) List(ctx context.Context, clientAny any, listPath string) (models.ListFilesResult, error) {
	client, ok := clientAny.(*gowebdav.Client)
	if !ok || client == nil {
		return models.ListFilesResult{}, transport.ProtocolError(errors.New("invalid webdav client"))
	}
	if strings.TrimSpace(listPath) == "" {
		listPath = "/"
	}
	items, err := client.ReadDir(listPath)
	if err != nil {
		if ctx.Err() != nil {
			return models.ListFilesResult{}, transport.TimeoutError(ctx.Err())
		}
		return models.ListFilesResult{}, classifyWebDAVError(err)
	}
	entries := make([]models.FileEntry, 0, len(items))
	for _, fi := range items {
		p := path.Join(listPath, fi.Name())
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

func (a *Adapter) MkdirAll(ctx context.Context, clientAny any, dirPath string) error {
	client, ok := clientAny.(*gowebdav.Client)
	if !ok || client == nil {
		return transport.ProtocolError(errors.New("invalid webdav client"))
	}
	dirPath = strings.TrimSpace(dirPath)
	if dirPath == "" || dirPath == "/" {
		return nil
	}
	clean := path.Clean(dirPath)
	parts := strings.Split(clean, "/")
	current := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		current = current + "/" + part
		_ = client.Mkdir(current, 0)
		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}
	}
	return nil
}

func (a *Adapter) Download(ctx context.Context, clientAny any, remotePath string, localPath string, onProgress func(written int64, total int64)) error {
	client, ok := clientAny.(*gowebdav.Client)
	if !ok || client == nil {
		return transport.ProtocolError(errors.New("invalid webdav client"))
	}
	remotePath = strings.TrimSpace(remotePath)
	localPath = strings.TrimSpace(localPath)
	if remotePath == "" || localPath == "" {
		return transport.ValidationError(errors.New("paths required"))
	}

	var total int64
	if st, err := client.Stat(remotePath); err == nil {
		total = st.Size()
	}

	rc, err := client.ReadStream(remotePath)
	if err != nil {
		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}
		return classifyWebDAVError(err)
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
			return classifyWebDAVError(rerr)
		}
	}
	return nil
}

func (a *Adapter) Upload(ctx context.Context, clientAny any, localPath string, remotePath string, onProgress func(written int64, total int64)) error {
	client, ok := clientAny.(*gowebdav.Client)
	if !ok || client == nil {
		return transport.ProtocolError(errors.New("invalid webdav client"))
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
		reader = &progressReadSeeker{f: src, total: total, cb: onProgress}
	}

	if err := client.WriteStream(remotePath, reader, 0); err != nil {
		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}
		return classifyWebDAVError(err)
	}
	return nil
}

func (a *Adapter) Remove(ctx context.Context, clientAny any, remotePath string, recursive bool) error {
	client, ok := clientAny.(*gowebdav.Client)
	if !ok || client == nil {
		return transport.ProtocolError(errors.New("invalid webdav client"))
	}
	remotePath = strings.TrimSpace(remotePath)
	if remotePath == "" || remotePath == "/" {
		return transport.ValidationError(errors.New("remotePath required"))
	}

	if !recursive {
		if err := client.Remove(remotePath); err != nil {
			if ctx.Err() != nil {
				return transport.TimeoutError(ctx.Err())
			}
			return classifyWebDAVError(err)
		}
		return nil
	}

	var removeDirRecursive func(p string) error
	removeDirRecursive = func(p string) error {
		items, err := client.ReadDir(p)
		if err != nil {
			if ctx.Err() != nil {
				return transport.TimeoutError(ctx.Err())
			}
			if rerr := client.Remove(p); rerr == nil {
				return nil
			}
			return classifyWebDAVError(err)
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
			if err := client.Remove(child); err != nil {
				if ctx.Err() != nil {
					return transport.TimeoutError(ctx.Err())
				}
				return classifyWebDAVError(err)
			}
			if ctx.Err() != nil {
				return transport.TimeoutError(ctx.Err())
			}
		}
		if err := client.Remove(p); err != nil {
			if ctx.Err() != nil {
				return transport.TimeoutError(ctx.Err())
			}
			return classifyWebDAVError(err)
		}
		return nil
	}

	if err := client.Remove(remotePath); err == nil {
		return nil
	}
	return removeDirRecursive(remotePath)
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

type progressReadSeeker struct {
	f     *os.File
	total int64
	cb    func(written int64, total int64)
	n     int64
}

func (p *progressReadSeeker) Read(b []byte) (int, error) {
	n, err := p.f.Read(b)
	if n > 0 {
		p.n += int64(n)
		if p.cb != nil {
			p.cb(p.n, p.total)
		}
	}
	return n, err
}

func (p *progressReadSeeker) Seek(offset int64, whence int) (int64, error) {
	pos, err := p.f.Seek(offset, whence)
	if err != nil {
		return pos, err
	}
	if offset == 0 && whence == 0 {
		p.n = 0
		if p.cb != nil {
			p.cb(0, p.total)
		}
	}
	return pos, nil
}
