package auth_test

import (
	"encoding/base64"
	"os"
	"testing"

	"github.com/monoposer/dataspan/internal/auth"
)

func TestVaultEncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	os.Setenv(auth.EnvVaultKey, base64.StdEncoding.EncodeToString(key))
	t.Cleanup(func() { os.Unsetenv(auth.EnvVaultKey) })

	v, err := auth.NewVaultFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	data := map[string]any{"username": "u", "password": "p"}
	payload, err := v.Encrypt(data)
	if err != nil {
		t.Fatal(err)
	}
	out, err := v.Decrypt(payload)
	if err != nil {
		t.Fatal(err)
	}
	if out["username"] != "u" || out["password"] != "p" {
		t.Fatalf("decrypted: %+v", out)
	}
}

func TestMergeCredentialIntoOptions(t *testing.T) {
	opts := map[string]any{"username": "old"}
	cred := map[string]any{"password": "secret", "username": "new"}
	merged := auth.MergeCredentialIntoOptions(opts, cred)
	if merged["password"] != "secret" {
		t.Fatalf("password not merged")
	}
	if merged["username"] != "new" {
		t.Fatalf("username should be overridden by credential")
	}
}
