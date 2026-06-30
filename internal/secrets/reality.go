package secrets

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"
)

const shortIDBytes = 8

func GenerateRealitySecret() (RealitySecret, error) {
	privateKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return RealitySecret{}, fmt.Errorf("generate reality x25519 key: %w", err)
	}

	shortIDRaw := make([]byte, shortIDBytes)
	if _, err := rand.Read(shortIDRaw); err != nil {
		return RealitySecret{}, fmt.Errorf("generate reality short_id: %w", err)
	}

	secret := RealitySecret{
		PrivateKey: base64.RawURLEncoding.EncodeToString(privateKey.Bytes()),
		PublicKey:  base64.RawURLEncoding.EncodeToString(privateKey.PublicKey().Bytes()),
		ShortIDs:   []string{hex.EncodeToString(shortIDRaw)},
		CreatedAt:  time.Now().Unix(),
	}
	if err := ValidateRealitySecret(secret); err != nil {
		return RealitySecret{}, err
	}

	return secret, nil
}

func EnsureRealitySecret(store Store) (RealitySecret, bool, error) {
	secret, err := store.LoadReality()
	if err == nil {
		if err := ValidateRealitySecret(secret); err != nil {
			return RealitySecret{}, false, fmt.Errorf("validate existing reality secret: %w", err)
		}
		return secret, false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return RealitySecret{}, false, err
	}

	secret, err = GenerateRealitySecret()
	if err != nil {
		return RealitySecret{}, false, err
	}
	if err := store.SaveReality(secret); err != nil {
		return RealitySecret{}, false, err
	}

	return secret, true, nil
}

func ValidateRealitySecret(secret RealitySecret) error {
	if secret.PrivateKey == "" {
		return errors.New("private_key is required")
	}
	if secret.PublicKey == "" {
		return errors.New("public_key is required")
	}
	if len(secret.ShortIDs) == 0 {
		return errors.New("at least one short_id is required")
	}
	for _, shortID := range secret.ShortIDs {
		if !isLowerHexShortID(shortID) {
			return fmt.Errorf("invalid short_id %q: must be 16 lowercase hex characters", shortID)
		}
	}
	return nil
}

func isLowerHexShortID(shortID string) bool {
	if len(shortID) != shortIDBytes*2 {
		return false
	}
	for _, ch := range shortID {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}
	return true
}
