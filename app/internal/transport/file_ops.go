package transport

import (
	"context"

	"app/internal/models"
)

type FileOps interface {
	List(ctx context.Context, client any, path string) (models.ListFilesResult, error)
	Download(ctx context.Context, client any, remotePath string, localPath string, onProgress func(written int64, total int64)) error
	Upload(ctx context.Context, client any, localPath string, remotePath string, onProgress func(written int64, total int64)) error
	MkdirAll(ctx context.Context, client any, path string) error
}
