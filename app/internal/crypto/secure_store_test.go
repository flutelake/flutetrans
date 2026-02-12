package crypto

import (
	"errors"
	"testing"
)

func TestSecureStoreDecryptWrongPassphraseReturnsErrDecryptFailed(t *testing.T) {
	plain := []byte("hello")

	store1 := NewSecureStore()
	store1.SetPassphrase("passphrase-1")
	env, err := store1.Encrypt(plain)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	store2 := NewSecureStore()
	store2.SetPassphrase("passphrase-2")
	_, err = store2.Decrypt(env)
	if err == nil {
		t.Fatalf("expected decrypt error")
	}
	if !errors.Is(err, ErrDecryptFailed) {
		t.Fatalf("expected ErrDecryptFailed, got: %v", err)
	}
}

