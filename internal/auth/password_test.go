package auth

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("admin123")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if !VerifyPassword("admin123", hash) {
		t.Fatal("expected password to verify")
	}
	if VerifyPassword("wrong-password", hash) {
		t.Fatal("wrong password must not verify")
	}
}

func TestHashPasswordRejectsShortPassword(t *testing.T) {
	if _, err := HashPassword("short"); err == nil {
		t.Fatal("expected short password to fail")
	}
}
