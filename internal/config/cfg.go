package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Db_url string `json:"db_url"`
}

const cfg_name string = ".gatorconfig.json"

func Get_cfg_path() (string, error) {
	Home_dir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(Home_dir, cfg_name), nil
}

func Read_cfg(Cfg_path string) (*Config, error) {
	cfg := Config{}
	Data, err := os.ReadFile(Cfg_path)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(Data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
