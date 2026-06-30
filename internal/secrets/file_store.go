package secrets

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const secretFileMode = 0600

type FileStore struct {
	dataDir string
}

var _ Store = (*FileStore)(nil)

func NewFileStore(dataDir string) *FileStore {
	return &FileStore{
		dataDir: dataDir,
	}
}

func (s *FileStore) LoadToken() (string, error) {
	path := s.tokenPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("token file %q does not exist: %w", path, err)
		}
		return "", fmt.Errorf("load token file %q: %w", path, err)
	}

	return strings.TrimSpace(string(data)), nil
}

func (s *FileStore) SaveToken(token string) error {
	if err := s.ensureDataDir(); err != nil {
		return err
	}

	path := s.tokenPath()
	if err := os.WriteFile(path, []byte(token), secretFileMode); err != nil {
		return fmt.Errorf("save token file %q: %w", path, err)
	}
	return nil
}

func (s *FileStore) LoadReality() (RealitySecret, error) {
	path := s.realityPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return RealitySecret{}, fmt.Errorf("reality secret file %q does not exist: %w", path, err)
		}
		return RealitySecret{}, fmt.Errorf("load reality secret file %q: %w", path, err)
	}

	var secret RealitySecret
	if err := json.Unmarshal(data, &secret); err != nil {
		return RealitySecret{}, fmt.Errorf("load reality secret file %q: invalid JSON: %w", path, err)
	}

	return secret, nil
}

func (s *FileStore) SaveReality(secret RealitySecret) error {
	if err := s.ensureDataDir(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(secret, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal reality secret: %w", err)
	}
	data = append(data, '\n')

	path := s.realityPath()
	if err := os.WriteFile(path, data, secretFileMode); err != nil {
		return fmt.Errorf("save reality secret file %q: %w", path, err)
	}
	return nil
}

func (s *FileStore) ensureDataDir() error {
	if err := os.MkdirAll(s.dataDir, 0700); err != nil {
		return fmt.Errorf("create secret data directory %q: %w", s.dataDir, err)
	}
	return nil
}

func (s *FileStore) tokenPath() string {
	return filepath.Join(s.dataDir, "token")
}

func (s *FileStore) realityPath() string {
	return filepath.Join(s.dataDir, "reality.json")
}
