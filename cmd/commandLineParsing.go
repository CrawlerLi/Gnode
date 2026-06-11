package main

import (
	"fmt"

	"github.com/CrawlerLi/Gnode/internal/config"
)

func CreatWalletParsing(args []string) (username string, role string, err error) {
	if len(args) < 2 || len(args) > 3 {
		return "", "", fmt.Errorf("too many or too less arguments")
	}
	if len(args) == 2 {
		username = args[1]
		role = ""
	}

	if len(args) == 3 {
		username = args[1]
		role = args[2]
	}

	return username, role, nil

}

func InitParsing(args []string) (minerAddress string, configFilePath string, err error) {
	if len(args) < 1 || len(args) > 3 {
		return "", "", fmt.Errorf("too many or too less arguments")
	}

	minerAddress = ""

	if len(args) >= 2 {
		configFilePath = args[1]
	}

	if len(args) == 3 {
		minerAddress = args[2]
	}

	return minerAddress, configFilePath, nil
}

func NodeParsing(args []string) (configFilePath string, err error) {
	if len(args) < 1 || len(args) > 2 {
		return "", fmt.Errorf("too many or too less arguments")
	}

	if len(args) == 2 {
		configFilePath = args[1]
	}

	return configFilePath, nil
}

func runConfigCommand(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: config <show|use|reset>")
	}

	switch args[1] {
	case "show":
		if len(args) != 2 {
			return fmt.Errorf("usage: config show")
		}
		path, err := config.ActiveNodeConfigPath()
		if err != nil {
			return fmt.Errorf("config show: %w", err)
		}
		fmt.Printf("active node config: %s\n", path)
		return nil

	case "use":
		if len(args) != 3 {
			return fmt.Errorf("usage: config use <configFilePath>")
		}
		if err := config.UseNodeConfig(args[2]); err != nil {
			return fmt.Errorf("config use: %w", err)
		}
		fmt.Printf("active node config set to: %s\n", args[2])
		return nil

	case "reset":
		if len(args) != 2 {
			return fmt.Errorf("usage: config reset")
		}
		if err := config.ResetNodeConfig(); err != nil {
			return fmt.Errorf("config reset: %w", err)
		}
		fmt.Printf("active node config reset to: %s\n", config.DefaultNodeConfigFile)
		return nil

	default:
		return fmt.Errorf("unknown config command %q", args[1])
	}
}
