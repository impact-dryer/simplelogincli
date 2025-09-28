package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoad_EnvOverrides(t *testing.T) {
	// Use temp XDG config dir so we don't affect real config
	dir := t.TempDir()
	if runtime.GOOS != "windows" {
		os.Setenv("XDG_CONFIG_HOME", dir)
		defer os.Unsetenv("XDG_CONFIG_HOME")
	}
	os.Setenv("SIMPLELOGIN_BASE_URL", "https://example.com")
	defer os.Unsetenv("SIMPLELOGIN_BASE_URL")
	os.Setenv("SIMPLELOGIN_API_KEY", "env-key")
	defer os.Unsetenv("SIMPLELOGIN_API_KEY")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.BaseConfig.BaseURL != "https://example.com" {
		t.Fatalf("BaseURL = %q, want https://example.com", cfg.BaseConfig.BaseURL)
	}
	if cfg.APIKey != "env-key" {
		t.Fatalf("APIKey = %q, want env-key", cfg.APIKey)
	}
}

func TestSaveAndLoad_File(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()
	if runtime.GOOS != "windows" {
		os.Setenv("XDG_CONFIG_HOME", dir)
		defer os.Unsetenv("XDG_CONFIG_HOME")
	}
	// Ensure env doesn't interfere
	os.Unsetenv("SIMPLELOGIN_BASE_URL")
	os.Unsetenv("SIMPLELOGIN_API_KEY")

	cfg := SecureConfig{APIKey: "file-key", BaseConfig: Config{BaseURL: "https://host"}}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	// Check file exists with expected path
	p, err := userConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("config file not created at %s", p)
		}
		t.Fatal(err)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.APIKey != "file-key" || loaded.BaseConfig.BaseURL != "https://host" {
		t.Fatalf("loaded = %#v, want api_key=file-key base_url=https://host", loaded)
	}
	// Ensure file under our tmp XDG config
	if filepath.Dir(p) != filepath.Join(dir, configDirName) {
		t.Fatalf("config dir = %s, want under %s", filepath.Dir(p), filepath.Join(dir, configDirName))
	}
}
