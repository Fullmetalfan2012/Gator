package config

import (
	"os"
	"fmt"
	"encoding/json"
)


const configFileName = "/.gatorconfig.json"

type Config struct {
	DbUrl 				string `json:"db_url"`
	CurrentUserName 	string `json:"current_user_name"`
}

func getConfigFilePath() (string, error) {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		return "", fmt.Errorf("HOME-Variable empty")
	}
	return homeDir + configFileName, nil
}

func Read() (Config, error) {
	filePath, err := getConfigFilePath()
	if err != nil {
		return Config{}, fmt.Errorf("Error getting config file path: %w", err)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return Config{}, fmt.Errorf("Error reading config file: %w", err)
	}
	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return Config{}, fmt.Errorf("Error unmarshaling config: %w", err)
	}
	return config, nil
}

func write(cfg Config) error {
	filePath, err := getConfigFilePath()
	if err != nil {
		return fmt.Errorf("Error getting config file path: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("Error marshaling config: %w", err)
	}
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("Error writing config: %w", err)
	}
	return nil
}

func (c *Config) SetUser(username string) error {
	c.CurrentUserName = username
	err := write(*c)
	if err != nil {
		return fmt.Errorf("Error setting username: %w", err)
	}
	return nil
}