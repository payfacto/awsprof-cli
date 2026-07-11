// Package sso implements the AWS IAM Identity Center (SSO) device-login flow
// and an aws-CLI-compatible token cache.
package sso

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Token is a cached SSO access token plus the fields the aws CLI stores.
type Token struct {
	AccessToken  string
	ExpiresAt    time.Time
	StartURL     string
	Region       string
	ClientID     string
	ClientSecret string
	RefreshToken string
}

// cacheJSON mirrors the aws CLI v2 sso/cache token file schema.
type cacheJSON struct {
	AccessToken  string `json:"accessToken"`
	ExpiresAt    string `json:"expiresAt"`
	StartURL     string `json:"startUrl,omitempty"`
	Region       string `json:"region,omitempty"`
	ClientID     string `json:"clientId,omitempty"`
	ClientSecret string `json:"clientSecret,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
}

const expiresLayout = "2006-01-02T15:04:05Z"

// CacheKey returns the lowercase SHA1 hex of the given session name or URL.
//
// SHA1 here is intentional and correct: it is NOT used as a security primitive.
// It reproduces the aws CLI v2 cache *filename* derivation
// (~/.aws/sso/cache/<sha1>.json) so awsprof and the aws CLI interoperate on the
// same cached token. A SAST tool that flags crypto/sha1 as "weak hashing" is a
// false positive in this use.
func CacheKey(sessionOrStartURL string) string {
	sum := sha1.Sum([]byte(sessionOrStartURL))
	return hex.EncodeToString(sum[:])
}

// CacheFilePath returns ~/.aws/sso/cache/<key>.json. It keys on the session
// name when set, else the start URL.
func CacheFilePath(session, startURL string) (string, error) {
	key := session
	if key == "" {
		key = startURL
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".aws", "sso", "cache", CacheKey(key)+".json"), nil
}

// WriteToken writes tok to path in aws-CLI-compatible JSON, creating parent
// directories. The write is atomic: it writes a sibling temp file (mode 0600)
// and renames it over path, so a crash mid-write cannot leave a half-written,
// corrupt cache that both awsprof and the aws CLI would then fail to parse.
func WriteToken(path string, tok Token) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	j := cacheJSON{
		AccessToken:  tok.AccessToken,
		ExpiresAt:    tok.ExpiresAt.UTC().Format(expiresLayout),
		StartURL:     tok.StartURL,
		Region:       tok.Region,
		ClientID:     tok.ClientID,
		ClientSecret: tok.ClientSecret,
		RefreshToken: tok.RefreshToken,
	}
	data, err := json.Marshal(j)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".sso-token-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Best-effort cleanup: a no-op once the rename below has consumed tmpName,
	// and removes the temp file on any early-return error path.
	defer func() { _ = os.Remove(tmpName) }()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
