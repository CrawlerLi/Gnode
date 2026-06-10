package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const DefaultNodeConfigFile = "./configs/default_node.json"
const DefaultCLIConfigFile = "./configs/cli_config.json"

type CLIConfig struct {
	Version          int    `json:"version"`
	ActiveNodeConfig string `json:"active_node_config"`
}

func DefaultCLIConfig() *CLIConfig {
	return &CLIConfig{
		Version:          CurrentVersion,
		ActiveNodeConfig: DefaultNodeConfigFile,
	}
}

func LoadCLIConfig() (*CLIConfig, error) {
	data, err := os.ReadFile(DefaultCLIConfigFile)
	if os.IsNotExist(err) {
		return DefaultCLIConfig(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("load cli config: read file: %w", err)
	}

	var cfg CLIConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("load cli config: decode json: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("load cli config: validate: %w", err)
	}

	return &cfg, nil
}

func SaveCLIConfig(cfg *CLIConfig) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("save cli config: validate: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("save cli config: encode json: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(DefaultCLIConfigFile), 0700); err != nil {
		return fmt.Errorf("save cli config: create config dir: %w", err)
	}

	if err := os.WriteFile(DefaultCLIConfigFile, data, 0600); err != nil {
		return fmt.Errorf("save cli config: write file: %w", err)
	}

	return nil
}

func ActiveNodeConfigPath() (string, error) {
	cfg, err := LoadCLIConfig()
	if err != nil {
		return "", err
	}

	return cfg.ActiveNodeConfig, nil
}

func UseNodeConfig(path string) error {
	if path == "" {
		return fmt.Errorf("node config path is empty")
	}

	if _, err := Load(path); err != nil {
		return fmt.Errorf("use node config: validate node config %s: %w", path, err)
	}

	return SaveCLIConfig(&CLIConfig{
		Version:          CurrentVersion,
		ActiveNodeConfig: path,
	})
}

func ResetNodeConfig() error {
	return SaveCLIConfig(DefaultCLIConfig())
}

func (cfg *CLIConfig) Validate() error {
	if cfg == nil {
		return fmt.Errorf("cli config is nil")
	}
	if cfg.Version != CurrentVersion {
		return fmt.Errorf("unsupported cli config version %d", cfg.Version)
	}
	if cfg.ActiveNodeConfig == "" {
		return fmt.Errorf("active_node_config is empty")
	}

	return nil
}
