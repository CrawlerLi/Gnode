package config

import (
	"encoding/json"
	"fmt"
	"os"
)

const CurrentVersion = 1

type NodeConfig struct {
	Version    int      `json:"version"`
	NodeID     string   `json:"node_id"`
	ListenAddr string   `json:"listen_addr"`
	ChainDB    string   `json:"chain_db"`
	WalletDB   string   `json:"wallet_db"`
	Peers      []string `json:"peers"`
}

func Load(path string) (*NodeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load config: read file: %w", err)
	}

	var cfg NodeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("load config: decode json: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("load config: validate: %w", err)
	}

	return &cfg, nil
}

func Save(path string, cfg *NodeConfig) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("save config: validate: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("save config: encode json: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("save config: write file: %w", err)
	}

	return nil
}

func (cfg *NodeConfig) Validate() error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if cfg.Version != CurrentVersion {
		return fmt.Errorf("unsupported config version %d", cfg.Version)
	}
	if cfg.NodeID == "" {
		return fmt.Errorf("node_id is empty")
	}
	if cfg.ListenAddr == "" {
		return fmt.Errorf("listen_addr is empty")
	}
	if cfg.ChainDB == "" {
		return fmt.Errorf("chain_db is empty")
	}
	if cfg.WalletDB == "" {
		return fmt.Errorf("wallet_db is empty")
	}

	return nil
}
