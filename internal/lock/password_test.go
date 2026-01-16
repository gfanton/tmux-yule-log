package lock

import (
	"strings"
	"testing"
)

func TestHashAndVerifyPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{
			name:     "regular password",
			password: "mysecretpassword",
		},
		{
			name:     "empty password",
			password: "",
		},
		{
			name:     "unicode password",
			password: "пароль日本語🔐",
		},
		{
			name:     "password with special chars",
			password: "p@$$w0rd!#%^&*()",
		},
		{
			name:     "long password",
			password: strings.Repeat("a", 1000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword([]byte(tt.password))
			if err != nil {
				t.Fatalf("HashPassword() error = %v", err)
			}

			// Verify correct password matches
			match, err := VerifyPassword([]byte(tt.password), hash)
			if err != nil {
				t.Fatalf("VerifyPassword() error = %v", err)
			}
			if !match {
				t.Error("VerifyPassword() should return true for correct password")
			}

			// Verify wrong password doesn't match
			wrongPassword := tt.password + "wrong"
			match, err = VerifyPassword([]byte(wrongPassword), hash)
			if err != nil {
				t.Fatalf("VerifyPassword() with wrong password error = %v", err)
			}
			if match {
				t.Error("VerifyPassword() should return false for wrong password")
			}
		})
	}
}

func TestHashPassword_UniqueHashes(t *testing.T) {
	password := []byte("samepassword")

	hash1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash1 == hash2 {
		t.Error("HashPassword() should produce different hashes due to random salt")
	}

	// Both should still verify correctly
	match, _ := VerifyPassword(password, hash1)
	if !match {
		t.Error("hash1 should verify correctly")
	}

	match, _ = VerifyPassword(password, hash2)
	if !match {
		t.Error("hash2 should verify correctly")
	}
}

func TestVerifyPassword_InvalidFormat(t *testing.T) {
	password := []byte("test")

	tests := []struct {
		name    string
		encoded string
		wantErr bool
	}{
		{
			name:    "wrong number of separators - too few",
			encoded: "$argon2id$v=19$m=19456",
			wantErr: true,
		},
		{
			name:    "wrong number of separators - too many",
			encoded: "$argon2id$v=19$m=19456,t=2,p=1$salt$hash$extra",
			wantErr: true,
		},
		{
			name:    "wrong algorithm prefix",
			encoded: "$bcrypt$v=19$m=19456,t=2,p=1$c2FsdA$aGFzaA",
			wantErr: true,
		},
		{
			name:    "invalid base64 in salt",
			encoded: "$argon2id$v=19$m=19456,t=2,p=1$!!!invalid!!!$aGFzaA",
			wantErr: true,
		},
		{
			name:    "invalid base64 in hash",
			encoded: "$argon2id$v=19$m=19456,t=2,p=1$c2FsdA$!!!invalid!!!",
			wantErr: true,
		},
		{
			name:    "empty string",
			encoded: "",
			wantErr: true,
		},
		{
			name:    "just separators",
			encoded: "$$$$$",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := VerifyPassword(password, tt.encoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyPassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHashPassword_Format(t *testing.T) {
	hash, err := HashPassword([]byte("test"))
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	// Check format: $argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>
	if !strings.HasPrefix(hash, "$argon2id$v=19$") {
		t.Errorf("hash should start with $argon2id$v=19$, got %s", hash)
	}

	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		t.Errorf("hash should have 6 parts separated by $, got %d", len(parts))
	}

	// parts[0] is empty (before first $)
	// parts[1] is "argon2id"
	// parts[2] is "v=19"
	// parts[3] is "m=19456,t=2,p=1"
	// parts[4] is base64 salt
	// parts[5] is base64 hash

	if parts[1] != "argon2id" {
		t.Errorf("algorithm should be argon2id, got %s", parts[1])
	}

	if parts[2] != "v=19" {
		t.Errorf("version should be v=19, got %s", parts[2])
	}

	if parts[3] != "m=19456,t=2,p=1" {
		t.Errorf("params should be m=19456,t=2,p=1, got %s", parts[3])
	}
}
