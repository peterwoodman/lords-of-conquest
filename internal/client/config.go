package client

import (
	"encoding/json"
	"os"
	"path/filepath"
)

var configProfile string

// SetProfile sets the config profile for multiple instances.
func SetProfile(profile string) {
	configProfile = profile
}

// Config holds client configuration.
type Config struct {
	// Connection settings
	LastServer string `json:"last_server"`

	// Player identity (persisted token for reconnecting)
	PlayerToken string `json:"player_token"`
	PlayerName  string `json:"player_name"`
	PlayerID    string `json:"player_id"`

	// UI preferences
	SoundEnabled bool    `json:"sound_enabled"`
	MusicVolume  float64 `json:"music_volume"`
	SFXVolume    float64 `json:"sfx_volume"`

	// Window geometry (remembered between sessions)
	WindowWidth  int `json:"window_width,omitempty"`
	WindowHeight int `json:"window_height,omitempty"`
	WindowX      int `json:"window_x,omitempty"`
	WindowY      int `json:"window_y,omitempty"`
}

// DefaultConfig returns a config with default values.
func DefaultConfig() *Config {
	return &Config{
		LastServer:   "localhost:30000",
		SoundEnabled: true,
		MusicVolume:  0.7,
		SFXVolume:    0.8,
	}
}

// LoadConfig loads config from the user's config directory.
func LoadConfig() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return DefaultConfig(), err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return DefaultConfig(), err
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		// Keep the unreadable file (it may hold the player token) instead of
		// letting the next Save overwrite it
		os.WriteFile(path+".bak", data, 0644)
		return DefaultConfig(), err
	}

	return cfg, nil
}

// Save saves the config to disk.
func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	// Write then rename so a crash mid-write can't leave a truncated config
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// configPath returns the path to the config file.
func configPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	filename := "config.json"
	if configProfile != "" {
		filename = "config-" + configProfile + ".json"
	}

	return filepath.Join(configDir, "lords-of-conquest", filename), nil
}
