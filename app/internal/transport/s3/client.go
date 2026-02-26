package s3

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"app/internal/models"
	"app/internal/transport"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Adapter struct{}

func NewAdapter() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Test(ctx context.Context, profile models.ConnectionProfile) (time.Duration, error) {
	started := time.Now()
	clientAny, err := a.Connect(ctx, profile)
	if err != nil {
		return 0, err
	}
	client := clientAny.(*minio.Client)
	ctx, cancel := context.WithTimeout(ctx, transport.EffectiveTimeout(ctx, 10*time.Second))
	defer cancel()

	bucket := strings.TrimSpace(profile.Path)
	if bucket == "" {
		return 0, transport.ValidationError(errors.New("bucket required"))
	}

	_, err = client.BucketExists(ctx, bucket)
	if err != nil {
		if ctx.Err() != nil {
			return 0, transport.TimeoutError(ctx.Err())
		}
		return 0, classifyS3RequestError(err)
	}
	return time.Since(started), nil
}

func (a *Adapter) Connect(ctx context.Context, profile models.ConnectionProfile) (any, error) {
	endpoint := strings.TrimSpace(profile.Host)
	if endpoint == "" {
		return nil, transport.ValidationError(errors.New("endpoint required"))
	}

	secure := true
	hostPort := endpoint
	if u, err := url.Parse(endpoint); err == nil && u.Scheme != "" {
		secure = strings.EqualFold(u.Scheme, "https")
		if u.Host != "" {
			hostPort = u.Host
		}
	}

	host := hostPort
	port := 0
	if h, p, err := net.SplitHostPort(hostPort); err == nil {
		host = h
		if parsed, err := net.LookupPort("tcp", p); err == nil {
			port = parsed
		} else {
			return nil, transport.ValidationError(errors.New("invalid endpoint port"))
		}
	} else {
		if secure {
			port = 443
		} else {
			port = 80
		}
	}

	if err := transport.CheckHostReachable(ctx, host, 10*time.Second); err != nil {
		return nil, err
	}
	preflightConn, err := transport.DialTCP(ctx, host, port, 10*time.Second)
	if err != nil {
		return nil, err
	}
	_ = preflightConn.Close()

	accessKeyId := ""
	secretAccessKey := ""
	if profile.Credentials != nil {
		accessKeyId = profile.Credentials["accessKeyId"]
		secretAccessKey = profile.Credentials["secretAccessKey"]
	}
	if accessKeyId == "" {
		return nil, transport.ValidationError(errors.New("accessKeyId required"))
	}
	if secretAccessKey == "" {
		return nil, transport.ValidationError(errors.New("secretAccessKey required"))
	}

	transportTimeout := transport.EffectiveTimeout(ctx, 10*time.Second)
	httpTransport := http.DefaultTransport.(*http.Transport).Clone()
	httpTransport.ResponseHeaderTimeout = transportTimeout

	options := &minio.Options{
		Creds:     credentials.NewStaticV4(accessKeyId, secretAccessKey, ""),
		Secure:    secure,
		Region:    stringFromMetadata(profile.Metadata, "region"),
		Transport: httpTransport,
	}
	client, err := minio.New(hostPort, options)
	if err != nil {
		return nil, transport.ProtocolError(err)
	}
	return client, nil
}

func classifyS3RequestError(err error) error {
	if err == nil {
		return transport.ProtocolError(errors.New("s3 request failed"))
	}
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return transport.TimeoutError(err)
	}
	var resp minio.ErrorResponse
	if errors.As(err, &resp) {
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			return transport.AuthError(err)
		}
		code := strings.ToLower(resp.Code)
		if strings.Contains(code, "accessdenied") || strings.Contains(code, "invalidaccesskeyid") || strings.Contains(code, "signaturedoesnotmatch") {
			return transport.AuthError(err)
		}
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "accessdenied") || strings.Contains(msg, "invalidaccesskeyid") || strings.Contains(msg, "signaturedoesnotmatch") {
		return transport.AuthError(err)
	}
	return transport.ProtocolError(err)
}

func (a *Adapter) Disconnect(_ context.Context, _ any) error {
	return nil
}

func stringFromMetadata(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func parseS3BucketAndKey(s string) (string, string) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "/")
	if s == "" {
		return "", ""
	}
	parts := strings.SplitN(s, "/", 2)
	bucket := strings.TrimSpace(parts[0])
	key := ""
	if len(parts) == 2 {
		key = strings.TrimPrefix(parts[1], "/")
	}
	return bucket, key
}

func (a *Adapter) List(ctx context.Context, clientAny any, listPath string) (models.ListFilesResult, error) {
	client, ok := clientAny.(*minio.Client)
	if !ok || client == nil {
		return models.ListFilesResult{}, transport.ProtocolError(errors.New("invalid s3 client"))
	}
	bucket, keyPrefix := parseS3BucketAndKey(listPath)
	if bucket == "" {
		return models.ListFilesResult{}, transport.ValidationError(errors.New("bucket required"))
	}
	prefix := keyPrefix
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	seenDirs := map[string]bool{}
	seenFiles := map[string]bool{}
	entries := make([]models.FileEntry, 0)

	for obj := range client.ListObjects(ctx, bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
		if obj.Err != nil {
			if ctx.Err() != nil {
				return models.ListFilesResult{}, transport.TimeoutError(ctx.Err())
			}
			return models.ListFilesResult{}, classifyS3RequestError(obj.Err)
		}
		rel := strings.TrimPrefix(obj.Key, prefix)
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" {
			continue
		}
		first, rest, hasRest := strings.Cut(rel, "/")
		if first == "" {
			continue
		}
		if hasRest {
			if !seenDirs[first] {
				seenDirs[first] = true
				entries = append(entries, models.FileEntry{
					Name:       first,
					Path:       bucket + "/" + prefix + first,
					IsDir:      true,
					Size:       0,
					ModifiedAt: 0,
				})
			}
			_ = rest
			continue
		}
		if seenFiles[first] {
			continue
		}
		seenFiles[first] = true
		entries = append(entries, models.FileEntry{
			Name:       first,
			Path:       bucket + "/" + prefix + first,
			IsDir:      false,
			Size:       obj.Size,
			ModifiedAt: obj.LastModified.UnixMilli(),
		})
	}

	resolved := bucket
	if keyPrefix != "" {
		resolved = bucket + "/" + strings.TrimSuffix(prefix, "/")
	}
	return models.ListFilesResult{Path: resolved, Entries: entries}, nil
}

func (a *Adapter) MkdirAll(_ context.Context, _ any, _ string) error {
	return nil
}

func (a *Adapter) Download(ctx context.Context, clientAny any, remotePath string, localPath string, onProgress func(written int64, total int64)) error {
	client, ok := clientAny.(*minio.Client)
	if !ok || client == nil {
		return transport.ProtocolError(errors.New("invalid s3 client"))
	}
	remotePath = strings.TrimSpace(remotePath)
	localPath = strings.TrimSpace(localPath)
	if remotePath == "" || localPath == "" {
		return transport.ValidationError(errors.New("paths required"))
	}

	bucket, key := parseS3BucketAndKey(remotePath)
	if bucket == "" || strings.TrimSpace(key) == "" {
		return transport.ValidationError(errors.New("invalid s3 path"))
	}

	var total int64
	if st, err := client.StatObject(ctx, bucket, key, minio.StatObjectOptions{}); err == nil {
		total = st.Size
	}

	obj, err := client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}
		return classifyS3RequestError(err)
	}
	defer obj.Close()

	dst, err := os.Create(localPath)
	if err != nil {
		return transport.ProtocolError(err)
	}
	defer dst.Close()

	buf := make([]byte, 256*1024)
	var written int64
	for {
		n, rerr := obj.Read(buf)
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
			return classifyS3RequestError(rerr)
		}
	}
	return nil
}

func (a *Adapter) Upload(ctx context.Context, clientAny any, localPath string, remotePath string, onProgress func(written int64, total int64)) error {
	client, ok := clientAny.(*minio.Client)
	if !ok || client == nil {
		return transport.ProtocolError(errors.New("invalid s3 client"))
	}
	remotePath = strings.TrimSpace(remotePath)
	localPath = strings.TrimSpace(localPath)
	if remotePath == "" || localPath == "" {
		return transport.ValidationError(errors.New("paths required"))
	}

	bucket, key := parseS3BucketAndKey(remotePath)
	if bucket == "" || strings.TrimSpace(key) == "" {
		return transport.ValidationError(errors.New("invalid s3 path"))
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

	_, err = client.PutObject(ctx, bucket, key, reader, total, minio.PutObjectOptions{})
	if err != nil {
		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}
		return classifyS3RequestError(err)
	}
	return nil
}

func (a *Adapter) Remove(ctx context.Context, clientAny any, remotePath string, recursive bool) error {
	client, ok := clientAny.(*minio.Client)
	if !ok || client == nil {
		return transport.ProtocolError(errors.New("invalid s3 client"))
	}
	remotePath = strings.TrimSpace(remotePath)
	if remotePath == "" {
		return transport.ValidationError(errors.New("remotePath required"))
	}

	bucket, key := parseS3BucketAndKey(remotePath)
	if bucket == "" || strings.TrimSpace(key) == "" {
		return transport.ValidationError(errors.New("invalid s3 path"))
	}

	if !recursive {
		if err := client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{}); err != nil {
			if ctx.Err() != nil {
				return transport.TimeoutError(ctx.Err())
			}
			return classifyS3RequestError(err)
		}
		return nil
	}

	prefix := key
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	for obj := range client.ListObjects(ctx, bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
		if obj.Err != nil {
			if ctx.Err() != nil {
				return transport.TimeoutError(ctx.Err())
			}
			return classifyS3RequestError(obj.Err)
		}
		if err := client.RemoveObject(ctx, bucket, obj.Key, minio.RemoveObjectOptions{}); err != nil {
			if ctx.Err() != nil {
				return transport.TimeoutError(ctx.Err())
			}
			return classifyS3RequestError(err)
		}
		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}
	}

	_ = client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
	_ = client.RemoveObject(ctx, bucket, prefix, minio.RemoveObjectOptions{})
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
