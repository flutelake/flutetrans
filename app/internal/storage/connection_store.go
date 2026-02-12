package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"app/internal/crypto"
	"app/internal/models"
)

type ConnectionStore struct {
	filePath string
	secure   *crypto.SecureStore
}

func NewConnectionStore(secure *crypto.SecureStore) (*ConnectionStore, error) {
	if secure == nil {
		return nil, errors.New("secure store is nil")
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Join(configDir, "flutetrans")
	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		return nil, err
	}
	return &ConnectionStore{
		filePath: filepath.Join(baseDir, "connections.json.enc"),
		secure:   secure,
	}, nil
}

func (s *ConnectionStore) HasEncryptedFile() (bool, error) {
	if s == nil {
		return false, errors.New("store is nil")
	}
	_, err := os.Stat(s.filePath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (s *ConnectionStore) Load() ([]models.ConnectionProfile, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.ConnectionProfile{}, nil
		}
		return nil, err
	}

	var env crypto.Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}

	plaintext, err := s.secure.Decrypt(&env)
	if err != nil {
		return nil, err
	}

	var profiles []models.ConnectionProfile
	if err := json.Unmarshal(plaintext, &profiles); err != nil {
		return nil, err
	}
	return profiles, nil
}

func (s *ConnectionStore) Save(profiles []models.ConnectionProfile) error {
	plaintext, err := json.Marshal(profiles)
	if err != nil {
		return err
	}

	env, err := s.secure.Encrypt(plaintext)
	if err != nil {
		return err
	}

	data, err := json.Marshal(env)
	if err != nil {
		return err
	}

	tmpPath := s.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.filePath)
}
