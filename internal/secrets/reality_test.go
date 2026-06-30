package secrets

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGenerateRealitySecret(t *testing.T) {
	secret, err := GenerateRealitySecret()
	if err != nil {
		t.Fatalf("GenerateRealitySecret() error = %v", err)
	}

	if secret.PrivateKey == "" {
		t.Fatal("PrivateKey is empty")
	}
	if secret.PublicKey == "" {
		t.Fatal("PublicKey is empty")
	}
	if len(secret.ShortIDs) == 0 {
		t.Fatal("ShortIDs is empty")
	}
	if !isLowerHexShortID(secret.ShortIDs[0]) {
		t.Fatalf("ShortIDs[0] = %q, want 16 lowercase hex characters", secret.ShortIDs[0])
	}
	if secret.CreatedAt <= 0 {
		t.Fatalf("CreatedAt = %d, want positive Unix timestamp", secret.CreatedAt)
	}

	privateKey, err := base64.RawURLEncoding.DecodeString(secret.PrivateKey)
	if err != nil {
		t.Fatalf("PrivateKey is not raw URL base64: %v", err)
	}
	if len(privateKey) != 32 {
		t.Fatalf("decoded PrivateKey length = %d, want 32", len(privateKey))
	}

	publicKey, err := base64.RawURLEncoding.DecodeString(secret.PublicKey)
	if err != nil {
		t.Fatalf("PublicKey is not raw URL base64: %v", err)
	}
	if len(publicKey) != 32 {
		t.Fatalf("decoded PublicKey length = %d, want 32", len(publicKey))
	}
}

func TestEnsureRealitySecretGeneratesWhenMissing(t *testing.T) {
	dataDir := t.TempDir()
	store := NewFileStore(dataDir)

	secret, generated, err := EnsureRealitySecret(store)
	if err != nil {
		t.Fatalf("EnsureRealitySecret() error = %v", err)
	}
	if !generated {
		t.Fatal("generated = false, want true")
	}
	if err := ValidateRealitySecret(secret); err != nil {
		t.Fatalf("generated secret invalid: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dataDir, "reality.json")); err != nil {
		t.Fatalf("Stat(reality.json) error = %v", err)
	}
}

func TestEnsureRealitySecretDoesNotRegenerateExisting(t *testing.T) {
	store := NewFileStore(t.TempDir())
	want := RealitySecret{
		PrivateKey: "existing-private-key",
		PublicKey:  "existing-public-key",
		ShortIDs:   []string{"0123456789abcdef"},
		CreatedAt:  1710000000,
	}
	if err := store.SaveReality(want); err != nil {
		t.Fatalf("SaveReality() error = %v", err)
	}

	got, generated, err := EnsureRealitySecret(store)
	if err != nil {
		t.Fatalf("EnsureRealitySecret() error = %v", err)
	}
	if generated {
		t.Fatal("generated = true, want false")
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("EnsureRealitySecret() = %#v, want %#v", got, want)
	}
}

func TestGeneratedRealityJSONCanBeLoadedByFileStore(t *testing.T) {
	store := NewFileStore(t.TempDir())

	generated, wasGenerated, err := EnsureRealitySecret(store)
	if err != nil {
		t.Fatalf("EnsureRealitySecret() error = %v", err)
	}
	if !wasGenerated {
		t.Fatal("wasGenerated = false, want true")
	}

	loaded, err := store.LoadReality()
	if err != nil {
		t.Fatalf("LoadReality() error = %v", err)
	}
	if !reflect.DeepEqual(loaded, generated) {
		t.Fatalf("LoadReality() = %#v, want %#v", loaded, generated)
	}
}
