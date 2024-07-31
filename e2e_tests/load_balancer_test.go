package e2e_tests

import (
	"net/http"
	"strconv"
	"testing"

	testUtils "github.com/tiny-loadbalancer/e2e_tests/test_utils"
	"github.com/tiny-loadbalancer/internal/config"
	"github.com/tiny-loadbalancer/internal/constants"
)

func TestInvalidStrategy(t *testing.T) {
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Fatalf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, "invalid-strategy")
	_, _, port, teardownSuite := testUtils.SetupSuite(t, []string{}, config, nil)
	defer teardownSuite(t)

	_, err = http.Get("http://localhost:" + strconv.Itoa(port))
	if err == nil {
		t.Fatalf("Expected loadbalancer to return connection refused, but got %s", err)
	}
}

func TestInvalidHealthcheck(t *testing.T) {
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Fatalf("Error getting free port for load balancer")
	}
	c := config.Config{
		Port:                port,
		Strategy:            constants.IPHashing,
		HealthCheckInterval: "invalid-healthcheck",
		Servers: []config.Server{
			{
				Weight: 1,
				Url:    "http://localhost:1",
			},
		},
	}
	_, _, port, teardownSuite := testUtils.SetupSuite(t, []string{}, c, nil)
	defer teardownSuite(t)

	_, err = http.Get("http://localhost:" + strconv.Itoa(port))
	if err == nil {
		t.Fatalf("Expected loadbalancer to return connection refused, but got %s", err)
	}
}

func TestInvalidServerUrl(t *testing.T) {
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Fatalf("Error getting free port for load balancer")
	}
	c := config.Config{
		Port:                port,
		Strategy:            constants.IPHashing,
		HealthCheckInterval: "5s",
		Servers: []config.Server{
			{
				Weight: 1,
				Url:    "invalid-url",
			},
		},
	}
	_, _, port, teardownSuite := testUtils.SetupSuite(t, []string{}, c, nil)
	defer teardownSuite(t)

	_, err = http.Get("http://localhost:" + strconv.Itoa(port))
	if err == nil {
		t.Fatalf("Expected loadbalancer to return connection refused, but got %s", err)
	}
}
