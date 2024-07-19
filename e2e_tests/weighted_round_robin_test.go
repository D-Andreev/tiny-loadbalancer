package e2e_tests

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	testUtils "github.com/tiny-loadbalancer/e2e_tests/test_utils"
	"github.com/tiny-loadbalancer/internal/constants"
)

func TestRoundRobinWeighted(t *testing.T) {
	ports := testUtils.GetFreePorts(t, 3)
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Errorf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, constants.WeightedRoundRobin)
	_, _, port, teardownSuite := testUtils.SetupSuite(t, ports, config, []int{5, 3, 2})
	defer teardownSuite(t)

	testCases := []testUtils.TestCase{
		// cycle 1
		{ExpectedBody: "Hello from server " + ports[0]}, //4
		{ExpectedBody: "Hello from server " + ports[1]}, // 2
		{ExpectedBody: "Hello from server " + ports[2]}, // 1,
		{ExpectedBody: "Hello from server " + ports[0]}, // 3
		{ExpectedBody: "Hello from server " + ports[1]}, // 1
		{ExpectedBody: "Hello from server " + ports[2]}, // 0
		{ExpectedBody: "Hello from server " + ports[0]}, // 2
		{ExpectedBody: "Hello from server " + ports[1]}, // 0
		{ExpectedBody: "Hello from server " + ports[0]}, // 1
		{ExpectedBody: "Hello from server " + ports[0]}, // 0
		// cycle 2
		{ExpectedBody: "Hello from server " + ports[0]}, //4
		{ExpectedBody: "Hello from server " + ports[1]}, // 2
		{ExpectedBody: "Hello from server " + ports[2]}, // 1,
		{ExpectedBody: "Hello from server " + ports[0]}, // 3
		{ExpectedBody: "Hello from server " + ports[1]}, // 1
		{ExpectedBody: "Hello from server " + ports[2]}, // 0
		{ExpectedBody: "Hello from server " + ports[0]}, // 2
		{ExpectedBody: "Hello from server " + ports[1]}, // 0
		{ExpectedBody: "Hello from server " + ports[0]}, // 1
		{ExpectedBody: "Hello from server " + ports[0]}, // 0
	}

	testUtils.AssertLoadBalancerResponse(t, testCases, port)
}

func TestRoundRobinWeightedNoServersAreStarted(t *testing.T) {
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Errorf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, constants.WeightedRoundRobin)
	_, _, port, teardownSuite := testUtils.SetupSuite(t, []string{}, config, []int{5, 3, 2})
	defer teardownSuite(t)

	res, _ := http.Get("http://localhost:" + strconv.Itoa(port))
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected service unavailable status code, got %d", res.StatusCode)
	}
}

func TestRoundRobinWeightedServerDiesAndComesBackOnline(t *testing.T) {
	ports := testUtils.GetFreePorts(t, 3)
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Errorf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, constants.WeightedRoundRobin)
	serverProcesses, _, port, teardownSuite := testUtils.SetupSuite(t, ports, config, []int{5, 3, 2})
	defer teardownSuite(t)

	testUtils.StopServer(serverProcesses[0])
	time.Sleep(time.Second * 2)

	testCases := []testUtils.TestCase{
		{ExpectedBody: "Hello from server " + ports[1]},
		{ExpectedBody: "Hello from server " + ports[2]},
		{ExpectedBody: "Hello from server " + ports[1]},
		{ExpectedBody: "Hello from server " + ports[2]},
	}

	testUtils.AssertLoadBalancerResponse(t, testCases, port)

	cmd := testUtils.StartServer(ports[0])
	serverProcesses[0] = cmd
	defer testUtils.StopServer(serverProcesses[0])
	time.Sleep(time.Second * 2)

	testCases = []testUtils.TestCase{
		{ExpectedBody: "Hello from server " + ports[0]},
		{ExpectedBody: "Hello from server " + ports[1]},
		{ExpectedBody: "Hello from server " + ports[0]},
		{ExpectedBody: "Hello from server " + ports[0]},
		{ExpectedBody: "Hello from server " + ports[0]},
		{ExpectedBody: "Hello from server " + ports[0]},

		{ExpectedBody: "Hello from server " + ports[0]}, //4
		{ExpectedBody: "Hello from server " + ports[1]}, // 2
		{ExpectedBody: "Hello from server " + ports[2]}, // 1,
		{ExpectedBody: "Hello from server " + ports[0]}, // 3
		{ExpectedBody: "Hello from server " + ports[1]}, // 1
		{ExpectedBody: "Hello from server " + ports[2]}, // 0
		{ExpectedBody: "Hello from server " + ports[0]}, // 2
		{ExpectedBody: "Hello from server " + ports[1]}, // 0
		{ExpectedBody: "Hello from server " + ports[0]}, // 1
		{ExpectedBody: "Hello from server " + ports[0]}, // 0
	}

	testUtils.AssertLoadBalancerResponse(t, testCases, port)
}

func TestRoundRobinWeightedRetryRequestTurnedOff(t *testing.T) {
	ports := testUtils.GetFreePorts(t, 3)
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Errorf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, constants.WeightedRoundRobin)
	config.HealthCheckInterval = "30s"
	config.RetryRequests = false

	serverProcesses, _, port, teardownSuite := testUtils.SetupSuite(t, ports, config, []int{5, 3, 2})
	defer teardownSuite(t)

	testUtils.StopServer(serverProcesses[0])

	res, _ := http.Get("http://localhost:" + strconv.Itoa(port))
	if res.StatusCode != http.StatusBadGateway {
		t.Errorf("Expected bad gateway status code, got %d", res.StatusCode)
	}
}

func TestRoundRobinWeightedRetryRequestTurnedOn(t *testing.T) {
	ports := testUtils.GetFreePorts(t, 3)
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Errorf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, constants.WeightedRoundRobin)
	config.HealthCheckInterval = "30s"
	config.RetryRequests = true

	serverProcesses, _, port, teardownSuite := testUtils.SetupSuite(t, ports, config, []int{5, 3, 2})
	defer teardownSuite(t)

	testUtils.StopServer(serverProcesses[0])

	testCases := []testUtils.TestCase{
		{ExpectedBody: "Hello from server " + ports[1]},
		{ExpectedBody: "Hello from server " + ports[2]},
		{ExpectedBody: "Hello from server " + ports[1]},
		{ExpectedBody: "Hello from server " + ports[2]},
	}

	testUtils.AssertLoadBalancerResponse(t, testCases, port)
}
