package sso

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCacheKey(t *testing.T) {
	in := "payfacto"
	// Golden digest computed once via: printf '%s' payfacto | sha1sum
	want := "74fe53db16b54998e49b82b38055998dbc43dc61"
	if got := CacheKey(in); got != want {
		t.Fatalf("CacheKey(%q) = %q, want %q", in, got, want)
	}
}

func TestCacheFilePath(t *testing.T) {
	const (
		session  = "payfacto"
		startURL = "https://x/start"
	)

	gotSession, err := CacheFilePath(session, startURL)
	if err != nil {
		t.Fatal(err)
	}
	wantSessionSuffix := filepath.Join(".aws", "sso", "cache", CacheKey(session)+".json")
	if !strings.HasSuffix(gotSession, wantSessionSuffix) {
		t.Fatalf("CacheFilePath(%q, %q) = %q, want suffix %q", session, startURL, gotSession, wantSessionSuffix)
	}

	gotURL, err := CacheFilePath("", startURL)
	if err != nil {
		t.Fatal(err)
	}
	wantURLSuffix := filepath.Join(".aws", "sso", "cache", CacheKey(startURL)+".json")
	if !strings.HasSuffix(gotURL, wantURLSuffix) {
		t.Fatalf("CacheFilePath(%q, %q) = %q, want suffix %q", "", startURL, gotURL, wantURLSuffix)
	}
}

// WriteToken must produce a file matching the aws CLI v2 cache schema, and its
// atomic write must not leave a temp file behind.
func TestWriteTokenSchema(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "tok.json")
	exp := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	in := Token{AccessToken: "abc", ExpiresAt: exp, StartURL: "https://x/start", Region: "us-east-1"}
	if err := WriteToken(p, in); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var j cacheJSON
	if err := json.Unmarshal(data, &j); err != nil {
		t.Fatalf("written cache is not valid JSON: %v", err)
	}
	if j.AccessToken != "abc" || j.StartURL != "https://x/start" || j.Region != "us-east-1" {
		t.Fatalf("schema mismatch: %+v", j)
	}
	if want := exp.Format(expiresLayout); j.ExpiresAt != want {
		t.Errorf("expiresAt = %q, want %q", j.ExpiresAt, want)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "tok.json" {
			t.Errorf("atomic write left an unexpected file behind: %s", e.Name())
		}
	}
}
