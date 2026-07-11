// Package profiles discovers and classifies AWS named profiles from the shared
// config and credentials files.
package profiles

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/ini.v1"
)

// Type is the kind of an AWS profile.
type Type int

const (
	TypeUnknown Type = iota
	TypeSSO
	TypeStatic
	TypeAssumeRole
	TypeProcess
)

// SSOConfig holds the SSO settings resolved for an SSO profile.
type SSOConfig struct {
	SessionName string
	StartURL    string
	Region      string
	AccountID   string
	RoleName    string
}

// Profile is a named AWS profile.
type Profile struct {
	Name string
	Type Type
	SSO  *SSOConfig
}

func configPath() string {
	if p := os.Getenv("AWS_CONFIG_FILE"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aws", "config")
}

func credentialsPath() string {
	if p := os.Getenv("AWS_SHARED_CREDENTIALS_FILE"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aws", "credentials")
}

// List discovers profiles from the standard AWS file locations.
func List() ([]Profile, error) {
	return ListFrom(configPath(), credentialsPath())
}

// ListFrom discovers profiles from explicit file paths. Missing files are
// treated as empty. Results are sorted by name.
func ListFrom(cfgPath, credPath string) ([]Profile, error) {
	cfg, err := loadINI(cfgPath)
	if err != nil {
		return nil, err
	}
	cred, err := loadINI(credPath)
	if err != nil {
		return nil, err
	}

	sessions := map[string]SSOConfig{}
	for _, s := range cfg.Sections() {
		if name, ok := strings.CutPrefix(s.Name(), "sso-session "); ok {
			sessions[name] = SSOConfig{
				StartURL: s.Key("sso_start_url").String(),
				Region:   s.Key("sso_region").String(),
			}
		}
	}

	cfgSecs := map[string]*ini.Section{}
	names := map[string]bool{}
	for _, s := range cfg.Sections() {
		switch {
		case s.Name() == ini.DefaultSection || strings.HasPrefix(s.Name(), "sso-session "):
			continue
		case s.Name() == "default":
			names["default"] = true
			cfgSecs["default"] = s
		default:
			if pn, ok := strings.CutPrefix(s.Name(), "profile "); ok {
				names[pn] = true
				cfgSecs[pn] = s
			}
		}
	}

	credSecs := map[string]*ini.Section{}
	for _, s := range cred.Sections() {
		if s.Name() == ini.DefaultSection {
			continue
		}
		names[s.Name()] = true
		credSecs[s.Name()] = s
	}

	out := make([]Profile, 0, len(names))
	for name := range names {
		p := Profile{Name: name}
		classify(&p, cfgSecs[name], credSecs[name], sessions)
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func classify(p *Profile, cfgSec, credSec *ini.Section, sessions map[string]SSOConfig) {
	get := func(key string) string {
		if cfgSec != nil {
			if v := cfgSec.Key(key).String(); v != "" {
				return v
			}
		}
		if credSec != nil {
			if v := credSec.Key(key).String(); v != "" {
				return v
			}
		}
		return ""
	}

	if session := get("sso_session"); session != "" {
		sc := SSOConfig{SessionName: session, AccountID: get("sso_account_id"), RoleName: get("sso_role_name")}
		if s, ok := sessions[session]; ok {
			sc.StartURL, sc.Region = s.StartURL, s.Region
		}
		p.Type, p.SSO = TypeSSO, &sc
		return
	}
	if url := get("sso_start_url"); url != "" {
		p.Type = TypeSSO
		p.SSO = &SSOConfig{
			StartURL:  url,
			Region:    get("sso_region"),
			AccountID: get("sso_account_id"),
			RoleName:  get("sso_role_name"),
		}
		return
	}
	switch {
	case get("credential_process") != "":
		p.Type = TypeProcess
	case get("role_arn") != "":
		p.Type = TypeAssumeRole
	case get("aws_access_key_id") != "":
		p.Type = TypeStatic
	default:
		p.Type = TypeUnknown
	}
}

func loadINI(path string) (*ini.File, error) {
	if _, err := os.Stat(path); err != nil {
		return ini.Empty(), nil
	}
	return ini.Load(path)
}
