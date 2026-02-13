package transport

import (
	"context"
	"time"

	"app/internal/models"
)

type Adapter interface {
	Test(ctx context.Context, profile models.ConnectionProfile) (time.Duration, error)
	Connect(ctx context.Context, profile models.ConnectionProfile) (any, error)
	Disconnect(ctx context.Context, client any) error
}
