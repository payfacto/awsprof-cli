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

const expirySkew = 60 * time.Second

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

// ReadToken loads a cached token from path.
func ReadToken(path string) (Token, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Token{}, err
	}
	var j cacheJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return Token{}, err
	}
	exp, err := time.Parse(expiresLayout, j.ExpiresAt)
	if err != nil {
		return Token{}, err
	}
	return Token{
		AccessToken:  j.AccessToken,
		ExpiresAt:    exp,
		StartURL:     j.StartURL,
		Region:       j.Region,
		ClientID:     j.ClientID,
		ClientSecret: j.ClientSecret,
		RefreshToken: j.RefreshToken,
	}, nil
}

// WriteToken writes tok to path in aws-CLI-compatible JSON, creating parents.
func WriteToken(path string, tok Token) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
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
	return os.WriteFile(path, data, 0o600)
}

// Valid reports whether the token is present and not within the expiry skew.
func (t Token) Valid(now time.Time) bool {
	if t.AccessToken == "" {
		return false
	}
	return t.ExpiresAt.After(now.Add(expirySkew))
}
