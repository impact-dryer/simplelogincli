package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
}

const DefaultBaseURL = "https://app.simplelogin.io"

const (
	configDirName  = "simplelogincli"
	configFileName = "config.json"
)

func userConfigFile() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, configDirName)
	return filepath.Join(dir, configFileName), nil
}

// Load reads config from file and applies environment overrides
func Load() (Config, error) {
	var cfg Config
	cfg.BaseURL = getenvDefault("SIMPLELOGIN_BASE_URL", DefaultBaseURL)
	if envKey := os.Getenv("SIMPLELOGIN_API_KEY"); envKey != "" {
		cfg.APIKey = envKey
	}
	path, err := userConfigFile()
	if err != nil {
		return cfg, err
	}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &cfg)
	}
	if envKey := os.Getenv("SIMPLELOGIN_API_KEY"); envKey != "" {
		cfg.APIKey = envKey
	}
	if envBase := os.Getenv("SIMPLELOGIN_BASE_URL"); envBase != "" {
		cfg.BaseURL = envBase
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	return cfg, nil
}

// Save writes config to file with 0600 permission
func Save(cfg Config) error {
	path, err := userConfigFile()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = f.Write(data)
	return err
}

func getenvDefault(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}
