package config

import (
	"encoding/json"
	"os"

	"github.com/tiny-loadbalancer/internal/constants"
)

type Server struct {
	Url    string `json:"url"`
	Weight int    `json:"weight"`
}

type Config struct {
	Port                int                `json:"port"`
	Servers             []Server           `json:"servers"`
	Strategy            constants.Strategy `json:"strategy"`
	HealthCheckInterval string             `json:"healthCheckInterval"`
	RetryRequests       bool               `json:"retryRequests"`
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
