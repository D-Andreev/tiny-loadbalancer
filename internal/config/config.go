package config

import (
	"encoding/json"
	"os"

	"github.com/tiny-loadbalancer/internal/constants"
)

type Config struct {
	Port       string             `json:"port"`
	ServerUrls []string           `json:"serverUrls"`
	Strategy   constants.Strategy `json:"strategy"`
}

func ReadConfig(path string) (*Config, error) {
	var config *Config

	bytes, err := os.ReadFile(path)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}
