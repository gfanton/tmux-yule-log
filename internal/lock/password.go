package lock

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/argon2"

	"yule-log/internal/xdg"
)

// ---- Argon2id Parameters (OWASP 2025 recommended)

const (
	argon2Time    = 2         // iterations
	argon2Memory  = 19 * 1024 // 19 MB
	argon2Threads = 1         // parallelism
	argon2KeyLen  = 32        // output length
	saltLen       = 16        // salt length
)

var (
	ErrNoPassword     = errors.New("no password configured")
	ErrInvalidFormat  = errors.New("invalid password file format")
	ErrPasswordExists = errors.New("password already configured")
)

// ---- Password Storage Format
// Format: $argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>

// HashPassword creates an Argon2id hash of the password.
// Returns the hash in PHC string format.
func HashPassword(password []byte) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}

	hash := argon2.IDKey(password, salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	hashB64 := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argon2Memory, argon2Time, argon2Threads, saltB64, hashB64)

	return encoded, nil
}

// VerifyPassword checks if the password matches the stored hash.
// Uses constant-time comparison to prevent timing attacks.
func VerifyPassword(password []byte, encoded string) (bool, error) {
	salt, storedHash, err := parseEncodedHash(encoded)
	if err != nil {
		return false, err
	}

	computedHash := argon2.IDKey(password, salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	return subtle.ConstantTimeCompare(storedHash, computedHash) == 1, nil
}

// parseEncodedHash extracts salt and hash from PHC string format.
func parseEncodedHash(encoded string) ([]byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return nil, nil, ErrInvalidFormat
	}

	if parts[1] != "argon2id" {
		return nil, nil, ErrInvalidFormat
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, fmt.Errorf("decoding salt: %w", err)
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, fmt.Errorf("decoding hash: %w", err)
	}

	return salt, hash, nil
}

// ---- Password File Operations

// SavePassword stores the password hash to the config file.
func SavePassword(password []byte) error {
	path, err := xdg.PasswordFile()
	if err != nil {
		return fmt.Errorf("getting password file path: %w", err)
	}

	hash, err := HashPassword(password)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	if err := os.WriteFile(path, []byte(hash+"\n"), 0600); err != nil {
		return fmt.Errorf("writing password file: %w", err)
	}

	return nil
}

// LoadPasswordHash reads the stored password hash from the config file.
func LoadPasswordHash() (string, error) {
	path, err := xdg.PasswordFile()
	if err != nil {
		return "", fmt.Errorf("getting password file path: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNoPassword
		}
		return "", fmt.Errorf("reading password file: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

// PasswordExists checks if a password has been configured.
func PasswordExists() bool {
	path, err := xdg.PasswordFile()
	if err != nil {
		return false
	}

	_, err = os.Stat(path)
	return err == nil
}

// RemovePassword deletes the stored password hash.
func RemovePassword() error {
	path, err := xdg.PasswordFile()
	if err != nil {
		return fmt.Errorf("getting password file path: %w", err)
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing password file: %w", err)
	}

	return nil
}

// CheckPassword verifies if the given password matches the stored hash.
func CheckPassword(password []byte) (bool, error) {
	hash, err := LoadPasswordHash()
	if err != nil {
		return false, err
	}

	return VerifyPassword(password, hash)
}
