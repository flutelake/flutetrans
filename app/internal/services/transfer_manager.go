package services

import (
	"context"
	"sync"
	"time"

	"app/internal/models"
	"app/internal/transport"

	"github.com/google/uuid"
)

type TransferManager struct {
	mu   sync.Mutex
	emit func(models.TransfersPayload)
	byID map[string]*models.TransferItem
}

func NewTransferManager() *TransferManager {
	return &TransferManager{byID: map[string]*models.TransferItem{}}
}

func (m *TransferManager) SetEmitter(emit func(models.TransfersPayload)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.emit = emit
}

func (m *TransferManager) snapshotLocked() []models.TransferItem {
	items := make([]models.TransferItem, 0, len(m.byID))
	for _, it := range m.byID {
		if it == nil {
			continue
		}
		items = append(items, *it)
	}
	return items
}

func (m *TransferManager) emitLocked() {
	if m.emit == nil {
		return
	}
	m.emit(models.TransfersPayload{Items: m.snapshotLocked()})
}

func (m *TransferManager) List() []models.TransferItem {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snapshotLocked()
}

func (m *TransferManager) StartDownload(session *models.ActiveSession, ops transport.FileOps, remotePath string, localPath string) (models.TransferItem, error) {
	if session == nil {
		return models.TransferItem{}, ErrSessionNotFound
	}
	if ops == nil {
		return models.TransferItem{}, transport.ProtocolError(context.Canceled)
	}

	now := time.Now().UnixMilli()
	item := &models.TransferItem{
		ID:         uuid.NewString(),
		SessionID:  session.SessionID,
		Protocol:   session.Protocol,
		Direction:  models.TransferDownload,
		LocalPath:  localPath,
		RemotePath: remotePath,
		StartedAt:  now,
		Status:     models.TransferRunning,
	}

	m.mu.Lock()
	m.byID[item.ID] = item
	m.emitLocked()
	m.mu.Unlock()

	go func(id string, client any) {
		ctx, cancel := context.WithCancel(context.Background())
		_ = cancel
		lastEmit := time.Time{}
		progress := func(written int64, total int64) {
			m.mu.Lock()
			current := m.byID[id]
			if current != nil {
				current.BytesTransferred = written
				if total > 0 {
					current.BytesTotal = total
				}
			}
			should := lastEmit.IsZero() || time.Since(lastEmit) > 200*time.Millisecond
			if should {
				lastEmit = time.Now()
				m.emitLocked()
			}
			m.mu.Unlock()
		}

		err := ops.Download(ctx, client, remotePath, localPath, progress)

		m.mu.Lock()
		current := m.byID[id]
		if current != nil {
			current.FinishedAt = time.Now().UnixMilli()
			if err != nil {
				current.Status = models.TransferFailed
				current.Error = err.Error()
			} else {
				current.Status = models.TransferCompleted
			}
			m.emitLocked()
		}
		m.mu.Unlock()
	}(item.ID, session.Client)

	return *item, nil
}

func (m *TransferManager) StartUpload(session *models.ActiveSession, ops transport.FileOps, localPath string, remotePath string) (models.TransferItem, error) {
	if session == nil {
		return models.TransferItem{}, ErrSessionNotFound
	}
	if ops == nil {
		return models.TransferItem{}, transport.ProtocolError(context.Canceled)
	}

	now := time.Now().UnixMilli()
	item := &models.TransferItem{
		ID:         uuid.NewString(),
		SessionID:  session.SessionID,
		Protocol:   session.Protocol,
		Direction:  models.TransferUpload,
		LocalPath:  localPath,
		RemotePath: remotePath,
		StartedAt:  now,
		Status:     models.TransferRunning,
	}

	m.mu.Lock()
	m.byID[item.ID] = item
	m.emitLocked()
	m.mu.Unlock()

	go func(id string, client any) {
		ctx, cancel := context.WithCancel(context.Background())
		_ = cancel
		lastEmit := time.Time{}
		progress := func(written int64, total int64) {
			m.mu.Lock()
			current := m.byID[id]
			if current != nil {
				current.BytesTransferred = written
				if total > 0 {
					current.BytesTotal = total
				}
			}
			should := lastEmit.IsZero() || time.Since(lastEmit) > 200*time.Millisecond
			if should {
				lastEmit = time.Now()
				m.emitLocked()
			}
			m.mu.Unlock()
		}

		err := ops.Upload(ctx, client, localPath, remotePath, progress)

		m.mu.Lock()
		current := m.byID[id]
		if current != nil {
			current.FinishedAt = time.Now().UnixMilli()
			if err != nil {
				current.Status = models.TransferFailed
				current.Error = err.Error()
			} else {
				current.Status = models.TransferCompleted
			}
			m.emitLocked()
		}
		m.mu.Unlock()
	}(item.ID, session.Client)

	return *item, nil
}
