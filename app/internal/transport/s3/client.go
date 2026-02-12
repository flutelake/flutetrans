package s3

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
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
