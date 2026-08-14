package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"sync"

	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidPassword = errors.New("invalid password")
	dummyOnce          sync.Once
	dummyHash          string
)

func dummyPasswordHash() string {
	dummyOnce.Do(func() {
		h, err := HashPassword("not-a-real-user")
		if err != nil {
			dummyHash = ""
			return
		}
		dummyHash = h
	})
	return dummyHash
}

// BurnLoginTiming runs a password hash so unknown usernames take about as long
// as known ones.
func BurnLoginTiming(password string) {
	_ = VerifyPassword(password, dummyPasswordHash())
}

// HashPassword hashes a password with argon2id.
func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	// Store salt+hash as hex: 16+32 = 48 bytes = 96 hex chars
	return hex.EncodeToString(salt) + hex.EncodeToString(hash), nil
}

// VerifyPassword compares password with stored hash (salt+hash hex: 32+64 chars).
func VerifyPassword(password, stored string) error {
	if len(stored) < 32+64 {
		return ErrInvalidPassword
	}
	saltHex := stored[:32]
	hashHex := stored[32:]
	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return ErrInvalidPassword
	}
	hash, err := hex.DecodeString(hashHex)
	if err != nil {
		return ErrInvalidPassword
	}
	got := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	if subtle.ConstantTimeCompare(got, hash) != 1 {
		return ErrInvalidPassword
	}
	return nil
}
