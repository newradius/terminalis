package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type AppConfig struct {
	FontSize           int    `json:"fontSize"`
	Theme              string `json:"theme"`
	DefaultPort        int    `json:"defaultPort"`
	DefaultUsername    string `json:"defaultUsername"`
	ConnectTimeout     int    `json:"connectTimeout"` // seconds
	ScrollbackLines    int    `json:"scrollbackLines"`
	TerminalBackground string `json:"terminalBackground"`
}

func DefaultConfig() AppConfig {
	return AppConfig{
		FontSize:           14,
		Theme:              "dark",
		DefaultPort:        22,
		DefaultUsername:    "root",
		ConnectTimeout:     30,
		ScrollbackLines:    10000,
		TerminalBackground: "#14142a",
	}
}

func Load(dataDir string) AppConfig {
	cfg := DefaultConfig()
	data, err := os.ReadFile(filepath.Join(dataDir, "config.json"))
	if err != nil {
		return cfg
	}
	json.Unmarshal(data, &cfg)
	return cfg
}

func Save(dataDir string, cfg AppConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dataDir, "config.json"), data, 0600)
}
