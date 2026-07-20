package main

import (
	"strings"
	"testing"
)

func TestServerStartCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		configFile   string
		databaseFile string
		imagesDir    string
		want         string
	}{
		{
			name:         "explicit images dir",
			configFile:   "config.yaml",
			databaseFile: "annotations.db",
			imagesDir:    "./images",
			want:         "rotulador -c config.yaml -d annotations.db -i ./images",
		},
		{
			name:         "empty images dir defaults to images",
			configFile:   "custom.yaml",
			databaseFile: "data.db",
			imagesDir:    "",
			want:         "rotulador -c custom.yaml -d data.db -i images",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := serverStartCommand(tt.configFile, tt.databaseFile, tt.imagesDir)
			if got != tt.want {
				t.Fatalf("serverStartCommand() = %q, want %q", got, tt.want)
			}
			// The annotator subcommand was removed; root is the server.
			if strings.Contains(got, "annotator") {
				t.Fatalf("server start command must not mention removed annotator subcommand: %q", got)
			}
		})
	}
}
