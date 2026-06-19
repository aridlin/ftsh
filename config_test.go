package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadEmbeddedManifest(t *testing.T) {
	entries, err := LoadManifest(ConfigSource{Root: embeddedDefaults, BasePath: "embedded/defaults", Name: "embedded"})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 10 {
		t.Fatalf("len(entries) = %d, want expanded embedded manifest", len(entries))
	}
}

func TestApplyEmbeddedConfigWritesProfileFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	var out bytes.Buffer
	err := ApplyConfigSource(
		ConfigSource{Root: embeddedDefaults, BasePath: "embedded/defaults", Name: "embedded"},
		"dev",
		ConfigApplyOptions{Stdout: &out, NonInteractive: true, BackupExisting: true},
	)
	if err != nil {
		t.Fatal(err)
	}

	for _, rel := range []string{".zshrc", ".config/nvim/init.lua", ".config/zellij/config.kdl"} {
		if _, err := os.Stat(filepath.Join(home, rel)); err != nil {
			t.Fatalf("expected %s to be written: %v", rel, err)
		}
	}
}

func TestApplyConfigBacksUpConflict(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	target := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	err := ApplyConfigSource(
		ConfigSource{Root: embeddedDefaults, BasePath: "embedded/defaults", Name: "embedded"},
		"server",
		ConfigApplyOptions{Stdout: &out, NonInteractive: true, BackupExisting: true},
	)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "FTSH embedded zsh baseline") {
		t.Fatalf("target was not replaced: %q", data)
	}
	backups, err := filepath.Glob(filepath.Join(home, ".ftsh", "backups", "*", ".zshrc"))
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 {
		t.Fatalf("backups = %#v, want one backup", backups)
	}
}
