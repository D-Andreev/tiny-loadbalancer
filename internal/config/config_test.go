package config

import (
	"testing"

	"github.com/tiny-loadbalancer/internal/constants"
)

func TestValidateStrategy(t *testing.T) {
	c := &Config{}
	testCases := []struct {
		id     int
		input  string
		output bool
	}{
		{
			id:     1,
			input:  "invalid-strategy",
			output: false,
		},
		{
			id:     2,
			input:  string(constants.RoundRobin),
			output: true,
		},
		{
			id:     3,
			input:  string(constants.WeightedRoundRobin),
			output: true,
		},
	}

	for _, testCase := range testCases {
		res := c.validateStrategy(testCase.input)
		if res != testCase.output {
			t.Fatalf("Test case %d: Expected %t, got %t", testCase.id, testCase.output, res)
		}
	}
}

func TestValidateHealthCheck(t *testing.T) {
	c := &Config{}

	testCases := []struct {
		id     int
		input  string
		output bool
	}{
		{
			id:     1,
			input:  "-1",
			output: false,
		},
		{
			id:     2,
			input:  "invalid-interval",
			output: false,
		},
		{
			id:     3,
			input:  "",
			output: false,
		},
		{
			id:     4,
			input:  "5s",
			output: true,
		},
		{
			id:     5,
			input:  "1h",
			output: true,
		},
		{
			id:     6,
			input:  "2h35m",
			output: true,
		},
	}

	for _, testCase := range testCases {
		res := c.validateHealthCheckInterval(testCase.input)
		if res != testCase.output {
			t.Fatalf("Test case %d: Expected %t, got %t", testCase.id, testCase.output, res)
		}
	}
}

func TestValidateServers(t *testing.T) {
	c := &Config{
		Servers: []Server{
			{
				Url:    "http://localhost:8080",
				Weight: 1,
			},
			{
				Url:    "http://localhost:8081",
				Weight: 2,
			},
			{
				Url:    "invalid-url",
				Weight: 3,
			},
		},
		Strategy:            constants.RoundRobin,
		HealthCheckInterval: "5s",
		Port:                123,
	}

	err := c.ValidateConfig(c)
	errMessage := "Key: 'Config.Servers[2].Url' Error:Field validation for 'Url' failed on the 'url' tag"
	if err.Error() != errMessage {
		t.Fatalf("Expected error for invalid server, got %s", err)
	}
}
