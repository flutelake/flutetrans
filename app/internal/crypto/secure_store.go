package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

var ErrLocked = errors.New("secure store locked")
var ErrInvalidEnvelope = errors.New("invalid envelope")
var ErrDecryptFailed = errors.New("decrypt failed")

type Envelope struct {
	Version    int    `json:"v"`
	KDF        string `json:"kdf"`
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ct"`
}

type SecureStore struct {
	passphrase []byte
}

func NewSecureStore() *SecureStore {
	return &SecureStore{}
}

func (s *SecureStore) SetPassphrase(passphrase string) {
	s.passphrase = []byte(passphrase)
}

func (s *SecureStore) IsUnlocked() bool {
	return len(s.passphrase) > 0
}

func (s *SecureStore) Encrypt(plaintext []byte) (*Envelope, error) {
	if !s.IsUnlocked() {
		return nil, ErrLocked
	}

	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	key := argon2.IDKey(s.passphrase, salt, 1, 64*1024, 4, 32)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	return &Envelope{
		Version:    1,
		KDF:        "argon2id",
		Salt:       base64.RawStdEncoding.EncodeToString(salt),
		Nonce:      base64.RawStdEncoding.EncodeToString(nonce),
		Ciphertext: base64.RawStdEncoding.EncodeToString(ciphertext),
	}, nil
}

func (s *SecureStore) Decrypt(env *Envelope) ([]byte, error) {
	if !s.IsUnlocked() {
		return nil, ErrLocked
	}
	if env == nil || env.Version != 1 || env.KDF != "argon2id" {
		return nil, ErrInvalidEnvelope
	}

	salt, err := base64.RawStdEncoding.DecodeString(env.Salt)
	if err != nil {
		return nil, ErrInvalidEnvelope
	}
	nonce, err := base64.RawStdEncoding.DecodeString(env.Nonce)
	if err != nil {
		return nil, ErrInvalidEnvelope
	}
	ciphertext, err := base64.RawStdEncoding.DecodeString(env.Ciphertext)
	if err != nil {
		return nil, ErrInvalidEnvelope
	}

	key := argon2.IDKey(s.passphrase, salt, 1, 64*1024, 4, 32)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, ErrInvalidEnvelope
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptFailed, err)
	}
	return plaintext, nil
}
