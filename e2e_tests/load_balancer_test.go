package e2e_tests

import (
	"net/http"
	"strconv"
	"testing"

	testUtils "github.com/tiny-loadbalancer/e2e_tests/test_utils"
)

func TestRoundRobinInvalidStrategy(t *testing.T) {
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Fatalf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, "invalid-strategy")
	_, _, port, teardownSuite := testUtils.SetupSuite(t, []string{}, config, nil)
	defer teardownSuite(t)

	res, err := http.Get("http://localhost:" + strconv.Itoa(port))
	if err != nil {
		t.Fatalf("Error sending request to load balancer: %s", err)
	}
	if err != nil {
		t.Fatalf("Error making GET request to load balancer: %v", err)
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected bad request status code, got %d", res.StatusCode)
	}
}

// TODO: Add tests for inalid config file, or other config props
