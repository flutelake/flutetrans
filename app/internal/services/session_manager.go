package services

import (
	"context"
	"errors"
	"sync"
	"time"

	"app/internal/models"
	"app/internal/transport"

	"github.com/google/uuid"
)

var ErrSessionNotFound = errors.New("session not found")

type StatusChangedPayload struct {
	SessionID string                  `json:"sessionID"`
	Status    models.ConnectionStatus `json:"status"`
	Message   string                  `json:"message"`
}

type SessionManager struct {
	mu       sync.RWMutex
	adapters map[models.ProtocolType]transport.Adapter
	sessions map[string]*models.ActiveSession
	current  string
	emit     func(StatusChangedPayload)
}

func NewSessionManager(adapters map[models.ProtocolType]transport.Adapter) *SessionManager {
	copyAdapters := map[models.ProtocolType]transport.Adapter{}
	for k, v := range adapters {
		copyAdapters[k] = v
	}
	return &SessionManager{adapters: copyAdapters, sessions: map[string]*models.ActiveSession{}}
}

func (m *SessionManager) SetEmitter(emit func(StatusChangedPayload)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.emit = emit
}

func (m *SessionManager) SetCurrent(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if sessionID == "connections" {
		m.current = ""
		return nil
	}
	if _, ok := m.sessions[sessionID]; !ok {
		return ErrSessionNotFound
	}
	m.current = sessionID
	return nil
}

func (m *SessionManager) Current() *models.ActiveSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.current == "" {
		return nil
	}
	return m.sessions[m.current]
}

func (m *SessionManager) Get(sessionID string) (*models.ActiveSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[sessionID]
	return s, ok
}

func (m *SessionManager) SetCurrentPath(sessionID string, currentPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[sessionID]
	if !ok || s == nil {
		return ErrSessionNotFound
	}
	s.CurrentPath = currentPath
	s.LastActivity = time.Now().UnixMilli()
	return nil
}

func (m *SessionManager) Adapter(protocol models.ProtocolType) (transport.Adapter, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	adapter, ok := m.adapters[protocol]
	return adapter, ok
}

func (m *SessionManager) StartConnect(profile models.ConnectionProfile) string {
	sessionID := uuid.NewString()

	ctx, cancel := context.WithCancel(context.Background())
	session := &models.ActiveSession{
		SessionID:    sessionID,
		ProfileID:    profile.ID,
		Protocol:     profile.Protocol,
		Status:       models.StatusConnecting,
		CurrentPath:  func() string { if profile.Protocol == models.ProtocolNFS { return "." }; return profile.Path }(),
		LastActivity: time.Now().UnixMilli(),
		CancelFunc:   cancel,
	}

	m.mu.Lock()
	m.sessions[sessionID] = session
	m.current = sessionID
	emit := m.emit
	adapter := m.adapters[profile.Protocol]
	m.mu.Unlock()

	if emit != nil {
		emit(StatusChangedPayload{SessionID: sessionID, Status: models.StatusConnecting, Message: "connecting"})
	}

	go func() {
		connectCtx, cancelTimeout := context.WithTimeout(ctx, 10*time.Second)
		defer cancelTimeout()
		client, err := adapter.Connect(connectCtx, profile)
		payload, ok := m.applyConnectResult(sessionID, client, err, connectCtx)
		if ok && emit != nil {
			emit(payload)
		}
	}()

	return sessionID
}

func (m *SessionManager) applyConnectResult(sessionID string, client any, err error, connectCtx context.Context) (StatusChangedPayload, bool) {
	m.mu.Lock()
	current := m.sessions[sessionID]
	if current == nil {
		m.mu.Unlock()
		return StatusChangedPayload{}, false
	}
	if connectCtx != nil {
		if ctxErr := connectCtx.Err(); ctxErr != nil && err == nil {
			err = ctxErr
		}
	}

	var payload StatusChangedPayload
	if err != nil {
		if errors.Is(connectCtx.Err(), context.DeadlineExceeded) {
			current.Status = models.StatusError
			payload = StatusChangedPayload{SessionID: sessionID, Status: models.StatusError, Message: "connection timeout"}
		} else if errors.Is(connectCtx.Err(), context.Canceled) {
			current.Status = models.StatusDisconnected
			payload = StatusChangedPayload{SessionID: sessionID, Status: models.StatusDisconnected, Message: "connection canceled"}
		} else {
			current.Status = models.StatusError
			payload = StatusChangedPayload{SessionID: sessionID, Status: models.StatusError, Message: err.Error()}
		}
		current.LastActivity = time.Now().UnixMilli()
		m.mu.Unlock()
		return payload, true
	}

	current.Client = client
	current.Status = models.StatusConnected
	current.LastActivity = time.Now().UnixMilli()
	payload = StatusChangedPayload{SessionID: sessionID, Status: models.StatusConnected, Message: "connected"}
	m.mu.Unlock()
	return payload, true
}

func (m *SessionManager) Disconnect(sessionID string) error {
	m.mu.Lock()
	session := m.sessions[sessionID]
	emit := m.emit
	if session == nil {
		m.mu.Unlock()
		return ErrSessionNotFound
	}
	message := "disconnected"
	if session.Status == models.StatusConnecting && session.Client == nil {
		message = "connection canceled"
	}
	adapter := m.adapters[session.Protocol]
	client := session.Client
	cancel := session.CancelFunc
	delete(m.sessions, sessionID)
	if m.current == sessionID {
		m.current = ""
	}
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	if client != nil && adapter != nil {
		dctx, dcancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = adapter.Disconnect(dctx, client)
		dcancel()
	}

	if emit != nil {
		emit(StatusChangedPayload{SessionID: sessionID, Status: models.StatusDisconnected, Message: message})
	}
	return nil
}
