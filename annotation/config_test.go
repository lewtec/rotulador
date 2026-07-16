package annotation

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestLoadConfig_HashesPlaintextPasswords(t *testing.T) {
	path := writeConfig(t, `
meta:
  description: test
auth:
  admin:
    password: "changeme"
tasks:
  - id: quality
    name: Quality
    classes:
      good:
        name: Good
`)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	stored := cfg.Authentication["admin"].Password
	if stored == "changeme" {
		t.Fatal("expected plaintext password to be hashed")
	}
	if !IsBcryptHash(stored) {
		t.Fatalf("stored password is not a bcrypt hash: %q", stored)
	}
	if !CheckPasswordHash("changeme", stored) {
		t.Fatal("hashed password does not verify original plaintext")
	}
}

func TestLoadConfig_HashesDollarTwoPlaintext(t *testing.T) {
	// Old prefix heuristic treated anything starting with "$2" as already hashed.
	plain := "$2secret"
	path := writeConfig(t, `
meta:
  description: test
auth:
  admin:
    password: "`+plain+`"
tasks:
  - id: quality
    name: Quality
    classes:
      good:
        name: Good
`)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	stored := cfg.Authentication["admin"].Password
	if stored == plain {
		t.Fatal("plaintext starting with $2 was left unhashed")
	}
	if !IsBcryptHash(stored) {
		t.Fatalf("stored password is not a bcrypt hash: %q", stored)
	}
	if !CheckPasswordHash(plain, stored) {
		t.Fatal("hashed password does not verify original $2-prefixed plaintext")
	}
}

func TestLoadConfig_PreservesExistingBcryptHash(t *testing.T) {
	hash, err := HashPassword("already-hashed")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	path := writeConfig(t, `
meta:
  description: test
auth:
  admin:
    password: "`+hash+`"
tasks:
  - id: quality
    name: Quality
    classes:
      good:
        name: Good
`)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	stored := cfg.Authentication["admin"].Password
	if stored != hash {
		t.Fatalf("existing bcrypt hash was rewritten\ngot  %s\nwant %s", stored, hash)
	}
	if !CheckPasswordHash("already-hashed", stored) {
		t.Fatal("preserved hash no longer verifies")
	}
}
