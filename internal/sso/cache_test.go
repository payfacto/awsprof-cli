package sso

import (
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

func TestWriteReadRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "tok.json")
	exp := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	in := Token{AccessToken: "abc", ExpiresAt: exp, StartURL: "https://x/start", Region: "us-east-1"}
	if err := WriteToken(p, in); err != nil {
		t.Fatal(err)
	}
	got, err := ReadToken(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "abc" || !got.ExpiresAt.Equal(exp) || got.Region != "us-east-1" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestTokenValid(t *testing.T) {
	now := time.Now()
	if (Token{AccessToken: "abc", ExpiresAt: now.Add(2 * time.Minute)}).Valid(now) != true {
		t.Errorf("token 2m out should be valid")
	}
	if (Token{AccessToken: "abc", ExpiresAt: now.Add(30 * time.Second)}).Valid(now) != false {
		t.Errorf("token inside 60s skew should be invalid")
	}
	if (Token{}).Valid(now) != false {
		t.Errorf("zero token should be invalid")
	}
	if (Token{AccessToken: "", ExpiresAt: now.Add(time.Hour)}).Valid(now) != false {
		t.Errorf("token with empty access token should be invalid")
	}
}
