package web

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

var errCloseFailed = errors.New("close failed")

type failCloser struct{}

func (failCloser) Close() error { return errCloseFailed }

type okCloser struct{}

func (okCloser) Close() error { return nil }

func TestReportClose(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	reportClose(t.Context(), okCloser{}, "msg", "failed to close config file")
	if buf.Len() != 0 {
		t.Fatalf("successful close was reported: %s", buf.String())
	}

	reportClose(t.Context(), failCloser{}, "msg", "failed to close config file", "path", "x")
	logged := buf.String()
	if !strings.Contains(logged, "close failed") || !strings.Contains(logged, "failed to close config file") || !strings.Contains(logged, "path=x") {
		t.Fatalf("close failure log = %q", logged)
	}
}
