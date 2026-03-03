package nfs

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

	"github.com/vmware/go-nfs-client/nfs"
	"github.com/vmware/go-nfs-client/nfs/rpc"
)

type Adapter struct{}

type conn struct {
	mount  *nfs.Mount
	target *nfs.Target
}

func NewAdapter() *Adapter {
	return &Adapter{}
}

func nfsCleanPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || p == "." || p == "/" {
		return "."
	}
	clean := path.Clean(p)
	if clean == "." || clean == "/" {
		return "."
	}
	clean = strings.TrimPrefix(clean, "/")
	if clean == "" {
		return "."
	}
	return clean
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

	if err := transport.CheckHostReachable(ctx, host, 10*time.Second); err != nil {
		return nil, err
	}
	port := profile.Port
	if port == 0 {
		port = 111
	}
	preflightConn, err := transport.DialTCP(ctx, host, port, 10*time.Second)
	if err != nil {
		return nil, err
	}
	_ = preflightConn.Close()

	exportPath := strings.TrimSpace(profile.Path)
	if exportPath == "" {
		return nil, transport.ValidationError(errors.New("exportPath required"))
	}

	addr := host
	if profile.Port != 0 {
		addr = net.JoinHostPort(host, fmt.Sprintf("%d", profile.Port))
	}

	timeout := transport.EffectiveTimeout(ctx, 10*time.Second)

	type result struct {
		mount  *nfs.Mount
		target *nfs.Target
		err    error
	}
	ch := make(chan result, 1)
	go func() {
		m, err := nfs.DialMount(addr)
		if err != nil {
			ch <- result{err: err}
			return
		}
		m.SetTimeout(timeout)
		t, err := m.Mount(exportPath, rpc.AuthNull)
		if err != nil {
			_ = m.Close()
			ch <- result{err: err}
			return
		}
		_, err = t.FSInfo()
		if err != nil {
			_ = t.Close()
			_ = m.Unmount()
			_ = m.Close()
			ch <- result{err: err}
			return
		}
		ch <- result{mount: m, target: t}
	}()

	select {
	case <-ctx.Done():
		return nil, transport.TimeoutError(ctx.Err())
	case r := <-ch:
		if r.err != nil {
			if errors.Is(r.err, context.DeadlineExceeded) || errors.Is(r.err, context.Canceled) {
				return nil, transport.TimeoutError(r.err)
			}
			return nil, transport.ProtocolError(r.err)
		}
		return &conn{mount: r.mount, target: r.target}, nil
	}
}

func (a *Adapter) Disconnect(_ context.Context, client any) error {
	c, ok := client.(*conn)
	if !ok || c == nil {
		return nil
	}
	if c.target != nil {
		_ = c.target.Close()
	}
	if c.mount != nil {
		_ = c.mount.Unmount()
		return c.mount.Close()
	}
	return nil
}

func (a *Adapter) List(ctx context.Context, clientAny any, listPath string) (models.ListFilesResult, error) {
	c, ok := clientAny.(*conn)
	if !ok || c == nil || c.target == nil {
		return models.ListFilesResult{}, transport.ProtocolError(errors.New("invalid nfs client"))
	}
	clean := nfsCleanPath(listPath)
	items, err := c.target.ReadDirPlus(clean)
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
		name := strings.TrimSpace(it.Name())
		if name == "" || name == "." || name == ".." {
			continue
		}
		p := path.Join(clean, name)
		if clean == "." {
			p = name
		}
		entries = append(entries, models.FileEntry{
			Name:       name,
			Path:       p,
			IsDir:      it.IsDir(),
			Size:       it.Size(),
			ModifiedAt: it.ModTime().UnixMilli(),
		})
	}
	return models.ListFilesResult{Path: clean, Entries: entries}, nil
}

func (a *Adapter) MkdirAll(ctx context.Context, clientAny any, dirPath string) error {
	c, ok := clientAny.(*conn)
	if !ok || c == nil || c.target == nil {
		return transport.ProtocolError(errors.New("invalid nfs client"))
	}
	clean := nfsCleanPath(dirPath)
	if clean == "." {
		return nil
	}

	parts := strings.Split(clean, "/")
	current := "."
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "." {
			continue
		}
		current = path.Join(current, part)
		if strings.HasPrefix(current, "./") {
			current = strings.TrimPrefix(current, "./")
		}

		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}

		if st, _, lookupErr := c.target.Lookup(current); lookupErr == nil {
			if st == nil || !st.IsDir() {
				return transport.ValidationError(errors.New("path exists and is not directory"))
			}
			continue
		}

		_, mkErr := c.target.Mkdir(current, 0755)
		if mkErr != nil {
			if st, _, lookupErr := c.target.Lookup(current); lookupErr == nil && st != nil && st.IsDir() {
				continue
			}
			if ctx.Err() != nil {
				return transport.TimeoutError(ctx.Err())
			}
			return transport.ProtocolError(mkErr)
		}
	}
	return nil
}

func (a *Adapter) Download(ctx context.Context, clientAny any, remotePath string, localPath string, onProgress func(written int64, total int64)) error {
	c, ok := clientAny.(*conn)
	if !ok || c == nil || c.target == nil {
		return transport.ProtocolError(errors.New("invalid nfs client"))
	}
	remotePath = strings.TrimSpace(remotePath)
	localPath = strings.TrimSpace(localPath)
	if remotePath == "" || localPath == "" {
		return transport.ValidationError(errors.New("paths required"))
	}
	remotePath = nfsCleanPath(remotePath)

	var total int64
	if st, _, err := c.target.Lookup(remotePath); err == nil && st != nil {
		total = st.Size()
	}

	src, err := c.target.Open(remotePath)
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
		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}
	}
	return nil
}

func (a *Adapter) Upload(ctx context.Context, clientAny any, localPath string, remotePath string, onProgress func(written int64, total int64)) error {
	c, ok := clientAny.(*conn)
	if !ok || c == nil || c.target == nil {
		return transport.ProtocolError(errors.New("invalid nfs client"))
	}
	remotePath = strings.TrimSpace(remotePath)
	localPath = strings.TrimSpace(localPath)
	if remotePath == "" || localPath == "" {
		return transport.ValidationError(errors.New("paths required"))
	}
	remotePath = nfsCleanPath(remotePath)

	src, err := os.Open(localPath)
	if err != nil {
		return transport.ProtocolError(err)
	}
	defer src.Close()

	var total int64
	if st, statErr := src.Stat(); statErr == nil {
		total = st.Size()
	}

	dst, err := c.target.OpenFile(remotePath, 0644)
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
				if ctx.Err() != nil {
					return transport.TimeoutError(ctx.Err())
				}
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
		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}
	}
	return nil
}

func (a *Adapter) Remove(ctx context.Context, clientAny any, remotePath string, recursive bool) error {
	c, ok := clientAny.(*conn)
	if !ok || c == nil || c.target == nil {
		return transport.ProtocolError(errors.New("invalid nfs client"))
	}
	remotePath = strings.TrimSpace(remotePath)
	if remotePath == "" || remotePath == "/" {
		return transport.ValidationError(errors.New("remotePath required"))
	}
	remotePath = nfsCleanPath(remotePath)

	if recursive {
		if err := c.target.RemoveAll(remotePath); err != nil {
			if ctx.Err() != nil {
				return transport.TimeoutError(ctx.Err())
			}
			return transport.ProtocolError(err)
		}
		return nil
	}

	if st, _, err := c.target.Lookup(remotePath); err == nil && st != nil && st.IsDir() {
		if err := c.target.RmDir(remotePath); err != nil {
			if ctx.Err() != nil {
				return transport.TimeoutError(ctx.Err())
			}
			return transport.ProtocolError(err)
		}
		return nil
	}

	if err := c.target.Remove(remotePath); err != nil {
		if ctx.Err() != nil {
			return transport.TimeoutError(ctx.Err())
		}
		return transport.ProtocolError(err)
	}
	return nil
}
