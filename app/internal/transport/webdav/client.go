package webdav

import (
	"context"
	"errors"
	"io"
	"net"
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
		reader = &progressReader{r: src, total: total, cb: onProgress}
	}

	if err := client.WriteStream(remotePath, reader, 0); err != nil {
		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}
		return classifyWebDAVError(err)
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
