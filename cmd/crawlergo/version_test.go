package main

import (
	"bytes"
	"runtime/debug"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestVersionInfo(t *testing.T) {
	originalVersion := version
	t.Cleanup(func() { version = originalVersion })

	t.Run("default version", func(t *testing.T) {
		version = "dev"
		got := formatVersion(version, func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{}, true
		})
		if want := "dev (commit unknown)"; got != want {
			t.Fatalf("formatVersion() = %q, want %q", got, want)
		}
	})

	t.Run("linker injected version", func(t *testing.T) {
		version = "v1.2.3"
		got := formatVersion(version, func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Settings: []debug.BuildSetting{{Key: "vcs.time", Value: "2026-11-26T10:00:00Z"}}}, true
		})
		if want := "v1.2.3 (commit 2026-11-26T10:00:00Z)"; got != want {
			t.Fatalf("formatVersion() = %q, want %q", got, want)
		}
	})
}

func TestVersionFlag(t *testing.T) {
	originalVersion := version
	originalPrinter := cli.VersionPrinter
	version = "test-version"
	cli.VersionPrinter = printVersion
	t.Cleanup(func() {
		version = originalVersion
		cli.VersionPrinter = originalPrinter
	})

	for _, flag := range []string{"--version", "-v"} {
		t.Run(flag, func(t *testing.T) {
			app := newApp()
			var output bytes.Buffer
			app.Writer = &output
			actionCalled := false
			app.Action = func(*cli.Context) error {
				actionCalled = true
				return nil
			}

			err := app.Run([]string{"crawlergo", flag})
			if err != nil {
				t.Fatalf("app.Run() error = %v, want nil", err)
			}
			if actionCalled {
				t.Fatal("version flag continued into the application action")
			}
			if got, want := output.String(), "test-version (commit unknown)\n"; got != want {
				t.Fatalf("output = %q, want %q", got, want)
			}
		})
	}
}

func TestVCSTimeUnavailable(t *testing.T) {
	tests := []struct {
		name string
		read func() (*debug.BuildInfo, bool)
	}{
		{name: "build info unavailable", read: func() (*debug.BuildInfo, bool) { return nil, false }},
		{name: "missing setting", read: func() (*debug.BuildInfo, bool) { return &debug.BuildInfo{}, true }},
		{name: "empty setting", read: func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Settings: []debug.BuildSetting{{Key: "vcs.time"}}}, true
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := vcsTime(tt.read); got != "unknown" {
				t.Fatalf("vcsTime() = %q, want %q", got, "unknown")
			}
		})
	}
}
