package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoad_MissingFileReturnsDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(cfg.Prefixes, []string{"payfacto-"}) {
		t.Fatalf("got %v, want [payfacto-]", cfg.Prefixes)
	}
}

func TestLoad_ReadsPrefixes(t *testing.T) {
	p := filepath.Join(t.TempDir(), "awsprof.yaml")
	if err := os.WriteFile(p, []byte("prefixes: [\"acme-\", \"corp-\"]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(cfg.Prefixes, []string{"acme-", "corp-"}) {
		t.Fatalf("got %v", cfg.Prefixes)
	}
}

func TestLoad_EmptyFileReturnsDefaults(t *testing.T) {
	p := filepath.Join(t.TempDir(), "awsprof.yaml")
	if err := os.WriteFile(p, []byte("\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(cfg.Prefixes, []string{"payfacto-"}) {
		t.Fatalf("got %v", cfg.Prefixes)
	}
}
