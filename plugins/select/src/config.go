package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type TLSConfig struct {
	Enabled bool   `json:"enabled"`
	CAFile  string `json:"caFile,omitempty"`
}

func (t *TLSConfig) UnmarshalJSON(b []byte) error {
	var raw struct {
		Enabled *bool           `json:"enabled"`
		CAFile  json.RawMessage `json:"caFile"`
	}
	if err := decodeJSON(b, &raw); err != nil {
		return err
	}
	if raw.Enabled == nil {
		return errors.New("TLS enabled must be a boolean")
	}
	t.Enabled = *raw.Enabled
	t.CAFile = ""
	if len(raw.CAFile) > 0 {
		if string(raw.CAFile) == "null" {
			return errors.New("invalid CA certificate path")
		}
		if err := json.Unmarshal(raw.CAFile, &t.CAFile); err != nil {
			return err
		}
		if t.CAFile == "" {
			return errors.New("invalid CA certificate path")
		}
	}
	return nil
}

type Connection struct {
	Host     string    `json:"host"`
	Port     int       `json:"port"`
	Database string    `json:"database"`
	User     string    `json:"user"`
	TLS      TLSConfig `json:"tls"`
	Password *string   `json:"password,omitempty"`
}

func (c *Connection) UnmarshalJSON(b []byte) error {
	type plain Connection
	v := plain{Port: 3306, TLS: TLSConfig{Enabled: true}}
	if err := decodeJSON(b, &v); err != nil {
		return err
	}
	*c = Connection(v)
	return nil
}

type Environment struct {
	Color      string      `json:"color,omitempty"`
	Connection *Connection `json:"connection,omitempty"`
}

type Project struct {
	Environments map[string]Environment `json:"environments"`
	Color        string                 `json:"color,omitempty"`
}
type Config struct {
	Version  int                `json:"version"`
	Projects map[string]Project `json:"projects"`
}
type legacyProject struct {
	Environments map[string]Connection `json:"environments"`
	Color        string                `json:"color,omitempty"`
}

type diskConfig struct {
	Version    int                   `json:"version"`
	Projects   json.RawMessage       `json:"projects,omitempty"`
	Profiles   map[string]Connection `json:"profiles,omitempty"`
	RuntimeDir string                `json:"runtimeDir,omitempty"`
}

var configMu sync.Mutex

func validName(s string) bool {
	return cleanText(s)
}
func validProjectColor(color string) bool {
	switch color {
	case "", "blue", "green", "purple", "orange", "red", "pink", "teal", "amber":
		return true
	}
	return false
}
func decodeJSON(b []byte, value any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return errors.New("unexpected trailing JSON")
	}
	return nil
}
func configDirectory(legacy bool) string {
	if !legacy {
		if p := os.Getenv("DB_PLUGIN_CONFIG_DIR"); p != "" {
			return absolute(p)
		}
		if p := os.Getenv("FALCON_DB_CONFIG_DIR"); p != "" {
			return absolute(p)
		}
	}
	home, _ := os.UserHomeDir()
	title := "DB Plugin"
	leaf := "db-plugin"
	if legacy {
		title = "Falcon DB"
		leaf = "falcon-db"
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", title)
	case "windows":
		p := os.Getenv("APPDATA")
		if p == "" {
			p = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(p, title)
	default:
		p := os.Getenv("XDG_CONFIG_HOME")
		if p == "" {
			p = filepath.Join(home, ".config")
		}
		return filepath.Join(p, leaf)
	}
}
func absolute(p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return a
}
func cleanText(s string) bool {
	if strings.TrimSpace(s) == "" {
		return false
	}
	for _, r := range s {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}
func validateConnection(c Connection) (Connection, error) {
	c.Host = strings.TrimSpace(c.Host)
	c.Database = strings.TrimSpace(c.Database)
	c.User = strings.TrimSpace(c.User)
	if !cleanText(c.Host) || strings.ContainsAny(c.Host, " /@?#\\\t\r\n") || !cleanText(c.Database) || !cleanText(c.User) || c.Port < 1 || c.Port > 65535 {
		return c, errors.New("check the host, port, database and username")
	}
	if c.TLS.CAFile != "" && (!filepath.IsAbs(c.TLS.CAFile) || !c.TLS.Enabled || !cleanText(c.TLS.CAFile)) {
		return c, errors.New("a CA certificate requires TLS and an absolute file path")
	}
	if c.Password != nil && strings.ContainsRune(*c.Password, 0) {
		return c, errors.New("password cannot contain a NUL character")
	}
	return c, nil
}
func loadConfig() (Config, error) {
	empty := Config{Version: 4, Projects: map[string]Project{}}
	p := filepath.Join(configDirectory(false), "profiles.json")
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) && os.Getenv("DB_PLUGIN_CONFIG_DIR") == "" && os.Getenv("FALCON_DB_CONFIG_DIR") == "" {
		b, err = os.ReadFile(filepath.Join(configDirectory(true), "profiles.json"))
	}
	if errors.Is(err, os.ErrNotExist) {
		return empty, nil
	}
	if err != nil {
		return empty, errors.New("could not read saved database settings")
	}
	if len(b) > 4<<20 {
		return empty, errors.New("configuration is too large")
	}
	var raw diskConfig
	if err = decodeJSON(b, &raw); err != nil {
		return empty, errors.New("invalid database configuration JSON")
	}
	var projects map[string]Project
	switch raw.Version {
	case 1, 2, 3:
		var legacy map[string]legacyProject
		if raw.Version == 1 {
			if raw.Projects != nil || raw.Profiles == nil {
				return empty, errors.New("invalid legacy configuration")
			}
			legacy = map[string]legacyProject{"falcon": {Environments: raw.Profiles}}
		} else if raw.Profiles != nil || decodeJSON(raw.Projects, &legacy) != nil || legacy == nil {
			return empty, errors.New("invalid project configuration")
		}
		projects = map[string]Project{}
		for name, old := range legacy {
			if old.Environments == nil {
				return empty, errors.New("invalid project environments")
			}
			p := Project{Color: old.Color, Environments: map[string]Environment{}}
			for env, connection := range old.Environments {
				p.Environments[env] = Environment{Color: defaultEnvironmentColor(env), Connection: &connection}
			}
			projects[name] = p
		}
	case 4:
		if raw.Profiles != nil || decodeJSON(raw.Projects, &projects) != nil || projects == nil {
			return empty, errors.New("invalid project configuration")
		}
	default:
		return empty, errors.New("unsupported configuration version")
	}
	if raw.RuntimeDir != "" && !filepath.IsAbs(raw.RuntimeDir) {
		return empty, errors.New("invalid legacy runtime directory")
	}
	for name, p := range projects {
		if !validName(name) || p.Environments == nil {
			return empty, errors.New("invalid project name or environments")
		}
		if !validProjectColor(p.Color) {
			return empty, errors.New("invalid project color")
		}
		for name, env := range p.Environments {
			if !validName(name) {
				return empty, errors.New("invalid environment name")
			}
			if env.Color == "" {
				env.Color = defaultEnvironmentColor(name)
			}
			if !validEnvironmentColor(env.Color) {
				return empty, errors.New("invalid environment color")
			}
			if env.Connection != nil {
				connection, err := validateConnection(*env.Connection)
				if err != nil {
					return empty, err
				}
				env.Connection = &connection
			}
			p.Environments[name] = env
		}
	}
	return Config{Version: 4, Projects: projects}, nil
}
func saveConfig(c Config) error {
	c.Version = 4
	dir := configDirectory(false)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return errors.New("could not create the configuration folder")
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if len(b) > 4<<20 {
		return errors.New("configuration is too large")
	}
	f, err := os.CreateTemp(dir, ".profiles-*")
	if err != nil {
		return errors.New("could not save configuration")
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(b)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(f.Name(), filepath.Join(dir, "profiles.json"))
	}
	if err != nil {
		return errors.New("could not save configuration")
	}
	return nil
}
func getConnection(project, environment string) (Connection, error) {
	if !validName(project) || !validName(environment) {
		return Connection{}, errors.New("select a valid project and environment")
	}
	c, err := loadConfig()
	if err != nil {
		return Connection{}, err
	}
	p, ok := c.Projects[project]
	if !ok {
		return Connection{}, errors.New("project is not configured")
	}
	v, ok := p.Environments[environment]
	if !ok || v.Connection == nil {
		return Connection{}, errors.New("environment is not configured")
	}
	return *v.Connection, nil
}
func statusData() (map[string]any, error) {
	c, err := loadConfig()
	if err != nil {
		return nil, err
	}
	p := map[string]any{}
	for name, v := range c.Projects {
		e := map[string]any{}
		for name, env := range v.Environments {
			e[name] = map[string]any{"configured": env.Connection != nil && env.Connection.Password != nil, "color": env.Color, "emoji": environmentEmoji(env.Color)}
		}
		p[name] = map[string]any{"environments": e}
	}
	return map[string]any{"projects": p}, nil
}
