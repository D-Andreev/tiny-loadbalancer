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
		t.Errorf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, "invalid-strategy")
	_, _, port, teardownSuite := testUtils.SetupSuite(t, []string{}, config, nil)
	defer teardownSuite(t)

	res, _ := http.Get("http://localhost:" + strconv.Itoa(port))
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected bad request status code, got %d", res.StatusCode)
	}
}

// TODO: Add tests for inalid config file, or other config props
