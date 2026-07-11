package sso

import (
	"crypto/sha1"
	"encoding/hex"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheKey(t *testing.T) {
	in := "payfacto"
	sum := sha1.Sum([]byte(in))
	want := hex.EncodeToString(sum[:])
	if got := CacheKey(in); got != want {
		t.Fatalf("CacheKey(%q) = %q, want %q", in, got, want)
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
	if (Token{ExpiresAt: now.Add(2 * time.Minute)}).Valid(now) != true {
		t.Errorf("token 2m out should be valid")
	}
	if (Token{ExpiresAt: now.Add(30 * time.Second)}).Valid(now) != false {
		t.Errorf("token inside 60s skew should be invalid")
	}
	if (Token{}).Valid(now) != false {
		t.Errorf("zero token should be invalid")
	}
}
