package annotation

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword securely hashes a password using bcrypt.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// IsBcryptHash reports whether s is a bcrypt hash bcrypt can parse (cost extractable).
// Prefer this over prefix heuristics like strings.HasPrefix(s, "$2"), which mis-detect
// plaintexts such as "$2secret" and leave users unable to log in.
func IsBcryptHash(s string) bool {
	_, err := bcrypt.Cost([]byte(s))
	return err == nil
}

// CheckPasswordHash compares a plaintext password with a hashed password.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// HashFile returns the SHA-256 hex digest of the file at path.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			ReportError(context.Background(), closeErr, "msg", "failed to close file after hashing", "path", path)
		}
	}()

	hasher := sha256.New()
	if _, err = io.Copy(hasher, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
