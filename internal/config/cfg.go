package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const cfg_name string = ".gatorconfig.json"

type Config struct {
	Db_url            string `json:"db_url"`
	Current_user_name string `json:"current_user_name"`
}

func (c *Config) SetUser(username string) {
	c.Current_user_name = username
}

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

func Write_cfg(cfg *Config) error {
	cfg_path, err := Get_cfg_path()
	if err != nil {
		return fmt.Errorf("Error in getting cfg path: %s", err)
	}
	Data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("CFG marshal error: %s", err)
	}
	if err := os.WriteFile(cfg_path, Data, 0644); err != nil {
		return fmt.Errorf("CFG write error: %s", err)
	}
	return nil
}
