package annotation

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.bin")
	content := []byte("rotulador-hash-fixture")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	got, err := HashFile(path)
	if err != nil {
		t.Fatalf("HashFile: %v", err)
	}

	want := fmt.Sprintf("%x", sha256.Sum256(content))
	if got != want {
		t.Fatalf("HashFile digest = %s, want %s", got, want)
	}
}

func TestHashFileMissing(t *testing.T) {
	_, err := HashFile(filepath.Join(t.TempDir(), "missing.bin"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestIsBcryptHash(t *testing.T) {
	hash, err := HashPassword("correct-horse")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "valid bcrypt", in: hash, want: true},
		{name: "plain password", in: "changeme", want: false},
		{name: "dollar-two prefix plaintext", in: "$2secret", want: false},
		{name: "short string", in: "ab", want: false},
		{name: "empty", in: "", want: false},
		{name: "truncated fake bcrypt", in: "$2a$10$notavalidbcrypthashvaluehere!!!", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBcryptHash(tt.in); got != tt.want {
				t.Fatalf("IsBcryptHash(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestCheckPasswordHash(t *testing.T) {
	hash, err := HashPassword("s3cret")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !CheckPasswordHash("s3cret", hash) {
		t.Fatal("expected password to match hash")
	}
	if CheckPasswordHash("wrong", hash) {
		t.Fatal("expected wrong password to fail")
	}
}
