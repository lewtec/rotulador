package annotation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecureJoin(t *testing.T) {
	// Create a temporary directory for valid base
	tmpDir, err := os.MkdirTemp("", "securejoin_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("failed to remove temp dir: %v", err)
		}
	}()

	// Get absolute path of tmpDir to ensure consistent comparison
	absBase, err := filepath.Abs(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		base      string
		file      string
		wantError bool
	}{
		{
			name:      "Valid file",
			base:      absBase,
			file:      "image.png",
			wantError: false,
		},
		{
			name:      "Traversal attempt",
			base:      absBase,
			file:      "../secret.txt",
			wantError: true,
		},
		{
			name:      "Nested traversal attempt",
			base:      absBase,
			file:      "images/../../secret.txt",
			wantError: true,
		},
		// Note: filepath.Join treats absolute paths as relative components usually,
		// but we should verify it doesn't escape.
		// On Unix, Join("dir", "/etc/passwd") -> "dir/etc/passwd".
		// But let's check what happens if it resolves to something outside.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := secureJoin(tt.base, tt.file)
			if (err != nil) != tt.wantError {
				t.Errorf("secureJoin() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError {
				// Verify the path is correct
				wantPath := filepath.Join(tt.base, tt.file)
				if got != wantPath {
					t.Errorf("secureJoin() = %v, want %v", got, wantPath)
				}
				// Verify it is inside base
				if !strings.HasPrefix(got, tt.base) {
					t.Errorf("secureJoin() result %v not inside base %v", got, tt.base)
				}
			}
		})
	}
}
