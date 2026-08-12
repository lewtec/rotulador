package web

import (
	"errors"
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

func TestLoadConfig_ValidationSentinels(t *testing.T) {
	cases := []struct {
		name string
		body string
		want error
	}{
		{
			name: "duplicate task id",
			body: `
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
  - id: quality
    name: Quality again
    classes:
      good:
        name: Good
`,
			want: ErrDuplicateTaskID,
		},
		{
			name: "task has no classes",
			body: `
meta:
  description: test
auth:
  admin:
    password: "changeme"
tasks:
  - id: quality
    name: Quality
    type: unknown-type
`,
			want: ErrTaskHasNoClasses,
		},
		{
			name: "no users",
			body: `
meta:
  description: test
tasks:
  - id: quality
    name: Quality
    classes:
      good:
        name: Good
`,
			want: ErrNoUsers,
		},
		{
			name: "i18n missing name",
			body: `
meta:
  description: test
auth:
  admin:
    password: "changeme"
i18n:
  - value: "hello"
tasks:
  - id: quality
    name: Quality
    classes:
      good:
        name: Good
`,
			want: ErrI18nMissingName,
		},
		{
			name: "i18n missing value",
			body: `
meta:
  description: test
auth:
  admin:
    password: "changeme"
i18n:
  - name: greeting
tasks:
  - id: quality
    name: Quality
    classes:
      good:
        name: Good
`,
			want: ErrI18nMissingValue,
		},
		{
			name: "null password",
			body: `
meta:
  description: test
auth:
  admin:
    password: ""
tasks:
  - id: quality
    name: Quality
    classes:
      good:
        name: Good
`,
			want: ErrNullPassword,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeConfig(t, tc.body)
			_, err := LoadConfig(path)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("LoadConfig: got %v, want %v", err, tc.want)
			}
		})
	}
}
