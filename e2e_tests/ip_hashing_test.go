package e2e_tests

import (
	"net/http"
	"strconv"
	"testing"

	testUtils "github.com/tiny-loadbalancer/e2e_tests/test_utils"
	"github.com/tiny-loadbalancer/internal/constants"
)

func TestIPHashing(t *testing.T) {
	ports := testUtils.GetFreePorts(t, 3)
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Errorf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, constants.IPHashing)
	_, _, port, teardownSuite := testUtils.SetupSuite(t, ports, config, nil)
	defer teardownSuite(t)

	// Requests from the same IP should always go to the same server
	testCases := []testUtils.TestCase{
		{ExpectedStatusCode: 200},
		{ExpectedStatusCode: 200},
		{ExpectedStatusCode: 200},
		{ExpectedStatusCode: 200},
		{ExpectedStatusCode: 200},
	}

	testUtils.AssertLoadBalancerStatusCode(t, testCases, port)
}

func TestIPHashingNoServersAreStarted(t *testing.T) {
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Errorf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, constants.IPHashing)
	_, _, port, teardownSuite := testUtils.SetupSuite(t, []string{}, config, nil)
	defer teardownSuite(t)

	res, _ := http.Get("http://localhost:" + strconv.Itoa(port))
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected service unavailable status code, got %d", res.StatusCode)
	}
}
