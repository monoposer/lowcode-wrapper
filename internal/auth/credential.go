package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

var ErrMasterKeyMissing = errors.New("WRAPPER_MASTER_KEY is required")

type Vault struct {
	aead cipher.AEAD
}

func NewVaultFromEnv() (*Vault, error) {
	keyB64 := os.Getenv("WRAPPER_MASTER_KEY")
	if keyB64 == "" {
		return nil, ErrMasterKeyMissing
	}
	key, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		return nil, fmt.Errorf("decode WRAPPER_MASTER_KEY: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("WRAPPER_MASTER_KEY must decode to 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Vault{aead: aead}, nil
}

func (v *Vault) Encrypt(data map[string]any) ([]byte, error) {
	if v == nil {
		return nil, errors.New("vault is nil")
	}
	plain, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, v.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return v.aead.Seal(nonce, nonce, plain, nil), nil
}

func (v *Vault) Decrypt(payload []byte) (map[string]any, error) {
	if v == nil {
		return nil, errors.New("vault is nil")
	}
	if len(payload) < v.aead.NonceSize() {
		return nil, errors.New("invalid credential payload")
	}
	nonce := payload[:v.aead.NonceSize()]
	ciphertext := payload[v.aead.NonceSize():]
	plain, err := v.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt credential: %w", err)
	}
	var out map[string]any
	if err := json.Unmarshal(plain, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// MergeCredentialIntoOptions overlays secret fields from credential data onto auth options.
func MergeCredentialIntoOptions(opts map[string]any, cred map[string]any) map[string]any {
	out := make(map[string]any, len(opts)+len(cred))
	for k, v := range opts {
		out[k] = v
	}
	secretKeys := []string{
		"username", "password", "apiKey", "token",
		"clientId", "clientSecret", "client_id", "client_secret",
	}
	for _, k := range secretKeys {
		if v, ok := cred[k]; ok && v != nil && fmt.Sprint(v) != "" {
			out[k] = v
		}
	}
	for k, v := range cred {
		if _, listed := findKey(secretKeys, k); !listed {
			if _, exists := out[k]; !exists {
				out[k] = v
			}
		}
	}
	return out
}

func findKey(keys []string, target string) (string, bool) {
	for _, k := range keys {
		if k == target {
			return k, true
		}
	}
	return "", false
}
