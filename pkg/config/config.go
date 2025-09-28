package config

import (
	"encoding/json"
	"github.com/zalando/go-keyring"
	"os"
	"path/filepath"
)

const service = "simplelogincli"
const user = "api_key"

type Config struct {
	BaseURL string `json:"base_url"`
}
type SecureConfig struct {
	BaseConfig Config `json:",inline"`
	APIKey     string `json:"api_key"`
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
func Load() (SecureConfig, error) {
	var cfg SecureConfig
	cfg.BaseConfig = Config{}
	cfg.BaseConfig.BaseURL = getenvDefault("SIMPLELOGIN_BASE_URL", DefaultBaseURL)

	// Try to get from keyring if not in env
	if cfg.APIKey == "" {
		if key, err := keyring.Get(service, user); err == nil {
			cfg.APIKey = key
		}
	}

	path, err := userConfigFile()
	if err != nil {
		return cfg, err
	}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &cfg.BaseConfig)
	}
	if envKey := os.Getenv("SIMPLELOGIN_API_KEY"); envKey != "" {
		cfg.APIKey = envKey
	}
	if envBase := os.Getenv("SIMPLELOGIN_BASE_URL"); envBase != "" {
		cfg.BaseConfig.BaseURL = envBase
	}
	if cfg.BaseConfig.BaseURL == "" {
		cfg.BaseConfig.BaseURL = DefaultBaseURL
	}
	return cfg, nil
}

// Save writes config to file with 0600 permission
func Save(cfg SecureConfig) error {
	path, err := userConfigFile()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg.BaseConfig, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = f.Write(data)

	if cfg.APIKey != "" {
		if err := keyring.Set(service, user, cfg.APIKey); err != nil {
			return err
		}
	}

	return err
}

func getenvDefault(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}
