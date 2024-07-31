package config

import (
	"encoding/json"
	"os"
	"time"

	"github.com/tiny-loadbalancer/internal/constants"
	"gopkg.in/go-playground/validator.v9"
)

type Server struct {
	Url    string `json:"url" validate:"required,url"`
	Weight int    `json:"weight"`
}

type Config struct {
	Port                int                `json:"port" validate:"gt=0"`
	Servers             []Server           `json:"servers" validate:"dive,required"`
	Strategy            constants.Strategy `json:"strategy" validate:"strategy"`
	HealthCheckInterval string             `json:"healthCheckInterval" validate:"healthCheckInterval"`
	RetryRequests       bool               `json:"retryRequests"`
}

func (c *Config) strategyValidatorFunc(fl validator.FieldLevel) bool {
	strategy := fl.Field().String()

	return c.validateStrategy(strategy)
}

func (c *Config) validateStrategy(strategy string) bool {
	for _, s := range constants.Strategies {
		if strategy == string(s) {
			return true
		}
	}

	return false
}

func (c *Config) healthCheckValidatorFunc(fl validator.FieldLevel) bool {
	interval := fl.Field().String()

	return c.validateHealthCheckInterval(interval)
}

func (c *Config) validateHealthCheckInterval(interval string) bool {
	_, err := time.ParseDuration(interval)
	if err != nil {
		return false
	}

	return true
}

func (c *Config) ReadConfig(path string) (*Config, error) {
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

func (c *Config) ValidateConfig(conf *Config) error {
	validate := validator.New()
	validate.RegisterValidation("strategy", c.strategyValidatorFunc)
	validate.RegisterValidation("healthCheckInterval", c.healthCheckValidatorFunc)

	err := validate.Struct(conf)
	if err != nil {
		return err
	}

	return nil
}
