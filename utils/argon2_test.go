package utils

import (
	"testing"
)

func TestArgon2Hashing(t *testing.T) {
	password := "my-very-secure-password"

	hash, err := GenerateArgon2Hash(password)
	if err != nil {
		t.Fatalf("failed to generate hash: %v", err)
	}

	if hash == "" {
		t.Fatal("generated hash is empty")
	}

	// Verify the hash matches
	match, err := CompareArgon2Hash(password, hash)
	if err != nil {
		t.Fatalf("failed to compare hash: %v", err)
	}
	if !match {
		t.Error("expected password to match its generated hash")
	}

	// Verify incorrect password doesn't match
	matchWrong, err := CompareArgon2Hash("wrong-password", hash)
	if err != nil {
		t.Fatalf("failed to compare wrong password: %v", err)
	}
	if matchWrong {
		t.Error("expected incorrect password to fail verification")
	}
}
