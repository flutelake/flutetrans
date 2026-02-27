package services

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"app/internal/crypto"
	"app/internal/models"
	"app/internal/storage"
	"app/internal/transport"
	ftpTransport "app/internal/transport/ftp"
	nfsTransport "app/internal/transport/nfs"
	s3Transport "app/internal/transport/s3"
	sftpTransport "app/internal/transport/sftp"
	smbTransport "app/internal/transport/smb"
	webdavTransport "app/internal/transport/webdav"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type ConnectionService struct {
	ctx       context.Context
	secure    *crypto.SecureStore
	store     *storage.ConnectionStore
	sessions  *SessionManager
	transfers *TransferManager
}

type MasterPasswordStatus struct {
	Unlocked          bool `json:"unlocked"`
	HasEncryptedStore bool `json:"hasEncryptedStore"`
}

func NewConnectionService() *ConnectionService {
	secure := crypto.NewSecureStore()
	manager := NewSessionManager(map[models.ProtocolType]transport.Adapter{
		models.ProtocolFTP:    ftpTransport.NewAdapter(),
		models.ProtocolSFTP:   sftpTransport.NewAdapter(),
		models.ProtocolS3:     s3Transport.NewAdapter(),
		models.ProtocolWebDAV: webdavTransport.NewAdapter(),
		models.ProtocolSMB:    smbTransport.NewAdapter(),
		models.ProtocolNFS:    nfsTransport.NewAdapter(),
	})
	return &ConnectionService{secure: secure, sessions: manager, transfers: NewTransferManager()}
}

func (s *ConnectionService) Startup(ctx context.Context) {
	s.ctx = ctx
	if s.sessions != nil {
		s.sessions.SetEmitter(func(payload StatusChangedPayload) {
			runtime.EventsEmit(ctx, "connection:status_changed", payload)
		})
	}
	if s.transfers != nil {
		s.transfers.SetEmitter(func(payload models.TransfersPayload) {
			runtime.EventsEmit(ctx, "transfer:updated", payload)
		})
	}
	if s.store == nil {
		store, err := storage.NewConnectionStore(s.secure)
		if err == nil {
			s.store = store
		}
	}
}

func (s *ConnectionService) SetMasterPassword(passphrase string) error {
	if strings.TrimSpace(passphrase) == "" {
		return validationError("master password required", nil)
	}
	s.secure.SetPassphrase(passphrase)
	if s.store == nil {
		store, err := storage.NewConnectionStore(s.secure)
		if err != nil {
			return storageError("failed to initialize store", map[string]any{"error": err.Error()})
		}
		s.store = store
	}
	return nil
}

func (s *ConnectionService) GetMasterPasswordStatus() (MasterPasswordStatus, error) {
	if s.store == nil {
		store, err := storage.NewConnectionStore(s.secure)
		if err != nil {
			return MasterPasswordStatus{}, storageError("failed to initialize store", map[string]any{"error": err.Error()})
		}
		s.store = store
	}

	has, err := s.store.HasEncryptedFile()
	if err != nil {
		return MasterPasswordStatus{}, storageError("failed to check encrypted store", map[string]any{"error": err.Error()})
	}

	return MasterPasswordStatus{Unlocked: s.secure.IsUnlocked(), HasEncryptedStore: has}, nil
}

func (s *ConnectionService) InitializeMasterPassword(passphrase string) error {
	if strings.TrimSpace(passphrase) == "" {
		return validationError("master password required", nil)
	}
	if err := s.SetMasterPassword(passphrase); err != nil {
		return err
	}
	if s.store == nil {
		return storageError("store not initialized", nil)
	}

	has, err := s.store.HasEncryptedFile()
	if err != nil {
		return storageError("failed to check encrypted store", map[string]any{"error": err.Error()})
	}
	if has {
		return validationError("master password already initialized", nil)
	}

	if err := s.store.Save([]models.ConnectionProfile{}); err != nil {
		if errors.Is(err, crypto.ErrLocked) {
			return cryptoError("master password not set", nil)
		}
		return storageError("failed to initialize encrypted store", map[string]any{"error": err.Error()})
	}
	return nil
}

func (s *ConnectionService) LockMasterPassword() error {
	s.secure.SetPassphrase("")
	return nil
}

func (s *ConnectionService) ListConnections() ([]models.ConnectionProfile, error) {
	profiles, err := s.loadAll()
	if err != nil {
		return nil, err
	}

	result := make([]models.ConnectionProfile, 0, len(profiles))
	for _, p := range profiles {
		result = append(result, sanitizeProfile(p))
	}
	return result, nil
}

func (s *ConnectionService) GetConnection(id string) (models.ConnectionProfile, error) {
	if strings.TrimSpace(id) == "" {
		return models.ConnectionProfile{}, validationError("id required", nil)
	}
	profiles, err := s.loadAll()
	if err != nil {
		return models.ConnectionProfile{}, err
	}
	for _, p := range profiles {
		if p.ID == id {
			return sanitizeProfile(p), nil
		}
	}
	return models.ConnectionProfile{}, validationError("not found", map[string]any{"id": id})
}

func (s *ConnectionService) SaveConnection(profile models.ConnectionProfile) (models.ConnectionProfile, error) {
	if err := validateProfile(profile); err != nil {
		return models.ConnectionProfile{}, err
	}

	profiles, err := s.loadAll()
	if err != nil {
		return models.ConnectionProfile{}, err
	}

	now := time.Now().UnixMilli()
	if strings.TrimSpace(profile.ID) == "" {
		profile.ID = uuid.NewString()
		profile.CreatedAt = now
		profile.UpdatedAt = now
		if profile.Credentials == nil {
			profile.Credentials = map[string]string{}
		}
		profiles = append(profiles, profile)
		if err := s.saveAll(profiles); err != nil {
			return models.ConnectionProfile{}, err
		}
		return sanitizeProfile(profile), nil
	}

	updated := false
	for i, existing := range profiles {
		if existing.ID != profile.ID {
			continue
		}

		merged := existing
		merged.Name = profile.Name
		merged.Protocol = profile.Protocol
		merged.Host = profile.Host
		merged.Port = profile.Port
		merged.AuthType = profile.AuthType
		merged.Path = profile.Path
		merged.Metadata = profile.Metadata
		merged.UpdatedAt = now

		merged.Credentials = mergeCredentials(existing.Credentials, profile.Credentials)
		profiles[i] = merged
		updated = true
		break
	}
	if !updated {
		return models.ConnectionProfile{}, validationError("not found", map[string]any{"id": profile.ID})
	}

	if err := s.saveAll(profiles); err != nil {
		return models.ConnectionProfile{}, err
	}
	return s.GetConnection(profile.ID)
}

func (s *ConnectionService) DeleteConnection(id string) error {
	if strings.TrimSpace(id) == "" {
		return validationError("id required", nil)
	}

	profiles, err := s.loadAll()
	if err != nil {
		return err
	}

	newProfiles := make([]models.ConnectionProfile, 0, len(profiles))
	deleted := false
	for _, p := range profiles {
		if p.ID == id {
			deleted = true
			continue
		}
		newProfiles = append(newProfiles, p)
	}
	if !deleted {
		return validationError("not found", map[string]any{"id": id})
	}

	return s.saveAll(newProfiles)
}

func (s *ConnectionService) TestConnection(profile models.ConnectionProfile) (map[string]any, error) {
	if err := validateProfile(profile); err != nil {
		return nil, err
	}
	if strings.TrimSpace(profile.ID) != "" {
		stored, err := s.loadProfileByID(profile.ID)
		if err == nil {
			if stored.Protocol == profile.Protocol && stored.AuthType == profile.AuthType {
				profile.Credentials = mergeCredentials(stored.Credentials, profile.Credentials)
			}
		}
	}
	logCtx := s.ctx
	if logCtx == nil {
		logCtx = context.Background()
	}
	LogInfo(logCtx, "connection.test", RedactedProfileFields(profile))

	adapter, ok := s.sessions.Adapter(profile.Protocol)
	if !ok {
		return nil, newServiceError(ErrCodeProtocol, "protocol not supported", map[string]any{"protocol": profile.Protocol})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	latency, err := adapter.Test(ctx, profile)
	if err != nil {
		LogError(logCtx, "connection.test_failed", map[string]any{"protocol": profile.Protocol, "error": err.Error()})
		return nil, s.mapTransportError(err)
	}
	LogInfo(logCtx, "connection.test_ok", map[string]any{"protocol": profile.Protocol, "latencyMs": latency.Milliseconds()})
	return map[string]any{"success": true, "message": "ok", "latencyMs": latency.Milliseconds()}, nil
}

func (s *ConnectionService) Connect(id string) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", validationError("id required", nil)
	}
	profile, err := s.loadProfileByID(id)
	if err != nil {
		return "", err
	}
	logCtx := s.ctx
	if logCtx == nil {
		logCtx = context.Background()
	}
	LogInfo(logCtx, "connection.connect", RedactedProfileFields(profile))

	if s.sessions == nil {
		return "", newServiceError(ErrCodeProtocol, "session manager not initialized", nil)
	}
	if _, ok := s.sessions.Adapter(profile.Protocol); !ok {
		return "", newServiceError(ErrCodeProtocol, "protocol not supported", map[string]any{"protocol": profile.Protocol})
	}
	sessionID := s.sessions.StartConnect(profile)
	LogInfo(logCtx, "connection.connect_started", map[string]any{"sessionID": sessionID, "protocol": profile.Protocol})
	return sessionID, nil
}

func (s *ConnectionService) Disconnect(sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return validationError("sessionID required", nil)
	}
	if s.sessions == nil {
		return newServiceError(ErrCodeProtocol, "session manager not initialized", nil)
	}
	if err := s.sessions.Disconnect(sessionID); err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return validationError("not found", map[string]any{"sessionID": sessionID})
		}
		return s.mapTransportError(err)
	}
	logCtx := s.ctx
	if logCtx == nil {
		logCtx = context.Background()
	}
	LogInfo(logCtx, "connection.disconnect", map[string]any{"sessionID": sessionID})
	return nil
}

func (s *ConnectionService) ListFiles(sessionID string, requestedPath string) (models.ListFilesResult, error) {
	if strings.TrimSpace(sessionID) == "" {
		return models.ListFilesResult{}, validationError("sessionID required", nil)
	}
	if s.sessions == nil {
		return models.ListFilesResult{}, newServiceError(ErrCodeProtocol, "session manager not initialized", nil)
	}
	session, ok := s.sessions.Get(sessionID)
	if !ok || session == nil {
		return models.ListFilesResult{}, validationError("not found", map[string]any{"sessionID": sessionID})
	}
	if session.Status != models.StatusConnected || session.Client == nil {
		return models.ListFilesResult{}, validationError("session not connected", map[string]any{"sessionID": sessionID})
	}
	adapter, ok := s.sessions.Adapter(session.Protocol)
	if !ok {
		return models.ListFilesResult{}, newServiceError(ErrCodeProtocol, "protocol not supported", map[string]any{"protocol": session.Protocol})
	}
	ops, ok := adapter.(transport.FileOps)
	if !ok {
		return models.ListFilesResult{}, newServiceError(ErrCodeProtocol, "file operations not supported", map[string]any{"protocol": session.Protocol})
	}

	listPath := strings.TrimSpace(requestedPath)
	if listPath == "" {
		listPath = strings.TrimSpace(session.CurrentPath)
	}
	if listPath == "" {
		listPath = "."
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := ops.List(ctx, session.Client, listPath)
	if err != nil {
		return models.ListFilesResult{}, s.mapTransportError(err)
	}
	_ = s.sessions.SetCurrentPath(sessionID, result.Path)
	return result, nil
}

func (s *ConnectionService) DeleteRemotePath(sessionID string, remotePath string, recursive bool) error {
	if strings.TrimSpace(sessionID) == "" {
		return validationError("sessionID required", nil)
	}
	remotePath = strings.TrimSpace(remotePath)
	if remotePath == "" {
		return validationError("remotePath required", nil)
	}
	if remotePath == "." || remotePath == "/" {
		return validationError("remotePath not allowed", map[string]any{"remotePath": remotePath})
	}
	if s.sessions == nil {
		return newServiceError(ErrCodeProtocol, "session manager not initialized", nil)
	}
	session, ok := s.sessions.Get(sessionID)
	if !ok || session == nil {
		return validationError("not found", map[string]any{"sessionID": sessionID})
	}
	if session.Status != models.StatusConnected || session.Client == nil {
		return validationError("session not connected", map[string]any{"sessionID": sessionID})
	}
	adapter, ok := s.sessions.Adapter(session.Protocol)
	if !ok {
		return newServiceError(ErrCodeProtocol, "protocol not supported", map[string]any{"protocol": session.Protocol})
	}
	ops, ok := adapter.(transport.FileOps)
	if !ok {
		return newServiceError(ErrCodeProtocol, "file operations not supported", map[string]any{"protocol": session.Protocol})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := ops.Remove(ctx, session.Client, remotePath, recursive); err != nil {
		return s.mapTransportError(err)
	}
	return nil
}

func (s *ConnectionService) CreateRemoteDir(sessionID string, dirPath string) error {
	if strings.TrimSpace(sessionID) == "" {
		return validationError("sessionID required", nil)
	}
	dirPath = strings.TrimSpace(dirPath)
	if dirPath == "" || dirPath == "." || dirPath == "/" {
		return validationError("dirPath required", map[string]any{"dirPath": dirPath})
	}
	if s.sessions == nil {
		return newServiceError(ErrCodeProtocol, "session manager not initialized", nil)
	}
	session, ok := s.sessions.Get(sessionID)
	if !ok || session == nil {
		return validationError("not found", map[string]any{"sessionID": sessionID})
	}
	if session.Status != models.StatusConnected || session.Client == nil {
		return validationError("session not connected", map[string]any{"sessionID": sessionID})
	}
	adapter, ok := s.sessions.Adapter(session.Protocol)
	if !ok {
		return newServiceError(ErrCodeProtocol, "protocol not supported", map[string]any{"protocol": session.Protocol})
	}
	ops, ok := adapter.(transport.FileOps)
	if !ok {
		return newServiceError(ErrCodeProtocol, "file operations not supported", map[string]any{"protocol": session.Protocol})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := ops.MkdirAll(ctx, session.Client, dirPath); err != nil {
		return s.mapTransportError(err)
	}
	return nil
}

func (s *ConnectionService) GetTransfers() ([]models.TransferItem, error) {
	if s.transfers == nil {
		return []models.TransferItem{}, nil
	}
	return s.transfers.List(), nil
}

func (s *ConnectionService) PickUploadFiles() ([]string, error) {
	if s.ctx == nil {
		return nil, newServiceError(ErrCodeProtocol, "runtime not initialized", nil)
	}
	paths, err := runtime.OpenMultipleFilesDialog(s.ctx, runtime.OpenDialogOptions{Title: "Select files"})
	if err != nil {
		return nil, newServiceError(ErrCodeProtocol, "dialog failed", map[string]any{"error": err.Error()})
	}
	return paths, nil
}

func (s *ConnectionService) StartDownload(sessionID string, remotePath string) (models.TransferItem, error) {
	if strings.TrimSpace(sessionID) == "" {
		return models.TransferItem{}, validationError("sessionID required", nil)
	}
	if strings.TrimSpace(remotePath) == "" {
		return models.TransferItem{}, validationError("remotePath required", nil)
	}
	if s.ctx == nil {
		return models.TransferItem{}, newServiceError(ErrCodeProtocol, "runtime not initialized", nil)
	}
	if s.sessions == nil {
		return models.TransferItem{}, newServiceError(ErrCodeProtocol, "session manager not initialized", nil)
	}
	session, ok := s.sessions.Get(sessionID)
	if !ok || session == nil {
		return models.TransferItem{}, validationError("not found", map[string]any{"sessionID": sessionID})
	}
	adapter, ok := s.sessions.Adapter(session.Protocol)
	if !ok {
		return models.TransferItem{}, newServiceError(ErrCodeProtocol, "protocol not supported", map[string]any{"protocol": session.Protocol})
	}
	ops, ok := adapter.(transport.FileOps)
	if !ok {
		return models.TransferItem{}, newServiceError(ErrCodeProtocol, "file operations not supported", map[string]any{"protocol": session.Protocol})
	}

	defaultName := path.Base(remotePath)
	localPath, err := runtime.SaveFileDialog(s.ctx, runtime.SaveDialogOptions{Title: "Save file", DefaultFilename: defaultName})
	if err != nil {
		return models.TransferItem{}, newServiceError(ErrCodeProtocol, "dialog failed", map[string]any{"error": err.Error()})
	}
	if strings.TrimSpace(localPath) == "" {
		return models.TransferItem{}, validationError("canceled", nil)
	}

	item, err := s.transfers.StartDownload(session, ops, remotePath, localPath)
	if err != nil {
		return models.TransferItem{}, s.mapTransportError(err)
	}
	return item, nil
}

func (s *ConnectionService) StartUpload(sessionID string, localPaths []string, remoteDir string) ([]models.TransferItem, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, validationError("sessionID required", nil)
	}
	if len(localPaths) == 0 {
		return nil, validationError("localPaths required", nil)
	}
	if s.sessions == nil {
		return nil, newServiceError(ErrCodeProtocol, "session manager not initialized", nil)
	}
	session, ok := s.sessions.Get(sessionID)
	if !ok || session == nil {
		return nil, validationError("not found", map[string]any{"sessionID": sessionID})
	}
	adapter, ok := s.sessions.Adapter(session.Protocol)
	if !ok {
		return nil, newServiceError(ErrCodeProtocol, "protocol not supported", map[string]any{"protocol": session.Protocol})
	}
	ops, ok := adapter.(transport.FileOps)
	if !ok {
		return nil, newServiceError(ErrCodeProtocol, "file operations not supported", map[string]any{"protocol": session.Protocol})
	}
	baseRemote := strings.TrimSpace(remoteDir)
	if baseRemote == "" {
		baseRemote = strings.TrimSpace(session.CurrentPath)
	}
	if baseRemote == "" {
		baseRemote = "."
	}

	items := make([]models.TransferItem, 0)

	for _, lp := range localPaths {
		lp = strings.TrimSpace(lp)
		if lp == "" {
			continue
		}
		st, err := os.Stat(lp)
		if err != nil {
			continue
		}
		if !st.IsDir() {
			remotePath := path.Join(baseRemote, filepath.Base(lp))
			_ = ops.MkdirAll(context.Background(), session.Client, path.Dir(remotePath))
			item, _ := s.transfers.StartUpload(session, ops, lp, remotePath)
			items = append(items, item)
			continue
		}

		root := lp
		_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if d == nil || d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			remotePath := path.Join(baseRemote, filepath.Base(root), rel)
			_ = ops.MkdirAll(context.Background(), session.Client, path.Dir(remotePath))
			item, _ := s.transfers.StartUpload(session, ops, p, remotePath)
			items = append(items, item)
			return nil
		})
	}

	return items, nil
}

func (s *ConnectionService) loadProfileByID(id string) (models.ConnectionProfile, error) {
	profiles, err := s.loadAll()
	if err != nil {
		return models.ConnectionProfile{}, err
	}
	for _, p := range profiles {
		if p.ID == id {
			return p, nil
		}
	}
	return models.ConnectionProfile{}, validationError("not found", map[string]any{"id": id})
}

func (s *ConnectionService) mapTransportError(err error) error {
	var te *transport.Error
	if errors.As(err, &te) {
		switch te.Kind {
		case transport.ErrorKindAuth:
			return newServiceError(ErrCodeAuth, "authentication failed", map[string]any{"error": te.Err.Error()})
		case transport.ErrorKindTimeout:
			return newServiceError(ErrCodeTimeout, "connection timeout", map[string]any{"error": te.Err.Error()})
		case transport.ErrorKindProtocol:
			return newServiceError(ErrCodeProtocol, "protocol error", map[string]any{"error": te.Err.Error()})
		case transport.ErrorKindValidation:
			return validationError(te.Err.Error(), nil)
		default:
			return newServiceError(ErrCodeProtocol, "connection failed", map[string]any{"error": te.Err.Error()})
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return newServiceError(ErrCodeTimeout, "connection timeout", map[string]any{"kind": "timeout"})
	}
	if errors.Is(err, context.Canceled) {
		return newServiceError(ErrCodeTimeout, "connection canceled", map[string]any{"kind": "canceled"})
	}
	return newServiceError(ErrCodeProtocol, "connection failed", map[string]any{"error": err.Error()})
}

func (s *ConnectionService) loadAll() ([]models.ConnectionProfile, error) {
	if s.store == nil {
		return nil, storageError("store not initialized", nil)
	}
	profiles, err := s.store.Load()
	if err != nil {
		if errors.Is(err, crypto.ErrLocked) {
			return nil, cryptoError("master password not set", nil)
		}
		if errors.Is(err, crypto.ErrDecryptFailed) {
			return nil, cryptoError("invalid master password", nil)
		}
		if errors.Is(err, crypto.ErrInvalidEnvelope) {
			return nil, cryptoError("invalid encrypted store", nil)
		}
		return nil, storageError("failed to load connections", map[string]any{"error": err.Error()})
	}
	sort.SliceStable(profiles, func(i, j int) bool {
		return profiles[i].UpdatedAt > profiles[j].UpdatedAt
	})
	return profiles, nil
}

func (s *ConnectionService) saveAll(profiles []models.ConnectionProfile) error {
	if s.store == nil {
		return storageError("store not initialized", nil)
	}
	if err := s.store.Save(profiles); err != nil {
		if err == crypto.ErrLocked {
			return cryptoError("master password not set", nil)
		}
		return storageError("failed to save connections", map[string]any{"error": err.Error()})
	}
	return nil
}

func validateProfile(p models.ConnectionProfile) error {
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return validationError("name required", nil)
	}
	if len(name) > 64 {
		return validationError("name too long", map[string]any{"max": 64})
	}
	if strings.TrimSpace(string(p.Protocol)) == "" {
		return validationError("protocol required", nil)
	}
	if p.Port < 0 || p.Port > 65535 {
		return validationError("invalid port", map[string]any{"port": p.Port})
	}
	return nil
}

func mergeCredentials(existing map[string]string, incoming map[string]string) map[string]string {
	merged := map[string]string{}
	for k, v := range existing {
		merged[k] = v
	}
	for k, v := range incoming {
		value := strings.TrimSpace(v)
		if value == "" {
			delete(merged, k)
			continue
		}
		merged[k] = v
	}
	return merged
}

func sanitizeProfile(p models.ConnectionProfile) models.ConnectionProfile {
	copy := p
	copy.Credentials = nil

	masked := map[string]bool{}
	for _, key := range sensitiveCredentialKeys(p.Protocol) {
		if p.Credentials != nil {
			if v, ok := p.Credentials[key]; ok && strings.TrimSpace(v) != "" {
				masked[key] = true
			}
		}
	}
	if len(masked) > 0 {
		copy.CredentialsMasked = masked
	}

	if p.Credentials != nil {
		sensitive := map[string]bool{}
		for _, key := range sensitiveCredentialKeys(p.Protocol) {
			sensitive[key] = true
		}

		safeCreds := map[string]string{}
		for k, v := range p.Credentials {
			if sensitive[k] {
				continue
			}
			value := strings.TrimSpace(v)
			if value == "" {
				continue
			}
			safeCreds[k] = v
		}
		if len(safeCreds) > 0 {
			copy.Credentials = safeCreds
		}
	}

	if p.Credentials != nil {
		if v := strings.TrimSpace(p.Credentials["username"]); v != "" {
			copy.Username = v
		} else if v := strings.TrimSpace(p.Credentials["accessKeyId"]); v != "" {
			copy.Username = v
		}
	}

	return copy
}

func sensitiveCredentialKeys(protocol models.ProtocolType) []string {
	switch protocol {
	case models.ProtocolFTP:
		return []string{"password"}
	case models.ProtocolSFTP:
		return []string{"password", "passphrase"}
	case models.ProtocolS3:
		return []string{"secretAccessKey"}
	case models.ProtocolWebDAV:
		return []string{"password"}
	case models.ProtocolSMB:
		return []string{"password"}
	case models.ProtocolNFS:
		return []string{}
	default:
		return []string{"password", "secretAccessKey", "passphrase"}
	}
}
