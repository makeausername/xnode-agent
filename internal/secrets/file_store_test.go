package secrets

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestFileStoreSaveTokenThenLoadToken(t *testing.T) {
	dataDir := t.TempDir()
	store := NewFileStore(dataDir)
	rawToken := "  node-token-123\n"

	if err := store.SaveToken(rawToken); err != nil {
		t.Fatalf("SaveToken() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dataDir, "token"))
	if err != nil {
		t.Fatalf("ReadFile(token) error = %v", err)
	}
	if string(data) != rawToken {
		t.Fatalf("token file = %q, want %q", string(data), rawToken)
	}

	token, err := store.LoadToken()
	if err != nil {
		t.Fatalf("LoadToken() error = %v", err)
	}
	if token != "node-token-123" {
		t.Fatalf("LoadToken() = %q, want %q", token, "node-token-123")
	}
}

func TestFileStoreSaveRealityThenLoadReality(t *testing.T) {
	dataDir := t.TempDir()
	store := NewFileStore(dataDir)
	want := RealitySecret{
		PrivateKey: "private-key",
		PublicKey:  "public-key",
		ShortIDs:   []string{"a1b2c3d4", "e5f6a7b8"},
		CreatedAt:  1710000000,
	}

	if err := store.SaveReality(want); err != nil {
		t.Fatalf("SaveReality() error = %v", err)
	}

	got, err := store.LoadReality()
	if err != nil {
		t.Fatalf("LoadReality() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadReality() = %#v, want %#v", got, want)
	}

	data, err := os.ReadFile(filepath.Join(dataDir, "reality.json"))
	if err != nil {
		t.Fatalf("ReadFile(reality.json) error = %v", err)
	}
	text := string(data)
	for _, field := range []string{"private_key", "public_key", "short_ids", "created_at"} {
		if !strings.Contains(text, `"`+field+`"`) {
			t.Fatalf("reality.json missing JSON field %q: %s", field, text)
		}
	}
	if !strings.Contains(text, "\n  \"private_key\"") {
		t.Fatalf("reality.json is not pretty JSON: %s", text)
	}
}

func TestFileStoreLoadTokenMissingReturnsError(t *testing.T) {
	store := NewFileStore(t.TempDir())

	_, err := store.LoadToken()
	if err == nil {
		t.Fatal("LoadToken() error = nil, want missing file error")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("LoadToken() error = %q, want missing file context", err.Error())
	}
}

func TestFileStoreLoadRealityInvalidJSONReturnsError(t *testing.T) {
	dataDir := t.TempDir()
	store := NewFileStore(dataDir)
	if err := os.WriteFile(filepath.Join(dataDir, "reality.json"), []byte("{invalid-json"), secretFileMode); err != nil {
		t.Fatalf("WriteFile(reality.json) error = %v", err)
	}

	_, err := store.LoadReality()
	if err == nil {
		t.Fatal("LoadReality() error = nil, want invalid JSON error")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Fatalf("LoadReality() error = %q, want invalid JSON context", err.Error())
	}
}

func TestFileStoreCreatesDataDirAutomatically(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "nested", "secrets")
	store := NewFileStore(dataDir)

	if err := store.SaveToken("node-token-123"); err != nil {
		t.Fatalf("SaveToken() error = %v", err)
	}

	info, err := os.Stat(dataDir)
	if err != nil {
		t.Fatalf("Stat(dataDir) error = %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("dataDir is not a directory")
	}
}
