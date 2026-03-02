package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Device struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Type    string `json:"type,omitempty"`
}

type Config struct {
	Devices []Device `json:"devices"`
}

func Load(path string) (Config, error) {
	var cfg Config
	f, err := os.Open(path)
	if err != nil {
		return cfg, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("decode config: %w", err)
	}

	if len(cfg.Devices) == 0 {
		return cfg, fmt.Errorf("no devices configured")
	}

	for i, d := range cfg.Devices {
		if d.Name == "" || d.Address == "" {
			return cfg, fmt.Errorf("device %d must include name and address", i)
		}
	}

	return cfg, nil
}

func Save(path string, cfg Config) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create config: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	return nil
}
