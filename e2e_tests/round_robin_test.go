package e2e_tests

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	testUtils "github.com/tiny-loadbalancer/e2e_tests/test_utils"
	"github.com/tiny-loadbalancer/internal/constants"
)

func TestRoundRobin(t *testing.T) {
	ports := testUtils.GetFreePorts(t, 3)
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Fatalf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, constants.RoundRobin)
	_, _, port, teardownSuite := testUtils.SetupSuite(t, ports, config, nil)
	defer teardownSuite(t)

	testCases := []testUtils.TestCase{
		{ExpectedBody: "Hello from server " + ports[0]},
		{ExpectedBody: "Hello from server " + ports[1]},
		{ExpectedBody: "Hello from server " + ports[2]},
		{ExpectedBody: "Hello from server " + ports[0]},
		{ExpectedBody: "Hello from server " + ports[1]},
		{ExpectedBody: "Hello from server " + ports[2]},
		{ExpectedBody: "Hello from server " + ports[0]},
	}

	testUtils.AssertLoadBalancerResponse(t, testCases, port)
}

func TestRoundRobinNoServersAreStarted(t *testing.T) {
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Fatalf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, constants.RoundRobin)
	_, _, port, teardownSuite := testUtils.SetupSuite(t, []string{}, config, nil)
	defer teardownSuite(t)

	res, err := http.Get("http://localhost:" + strconv.Itoa(port))
	if err != nil {
		t.Fatalf("Error sending request to load balancer: %s", err)
	}
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("Expected service unavailable status code, got %d", res.StatusCode)
	}
}

func TestRoundRobinServerDiesAndComesBackOnline(t *testing.T) {
	ports := testUtils.GetFreePorts(t, 3)
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Fatalf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, constants.RoundRobin)
	serverProcesses, _, port, teardownSuite := testUtils.SetupSuite(t, ports, config, nil)
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
		{ExpectedBody: "Hello from server " + ports[2]},
		{ExpectedBody: "Hello from server " + ports[0]},
		{ExpectedBody: "Hello from server " + ports[1]},
		{ExpectedBody: "Hello from server " + ports[2]},
		{ExpectedBody: "Hello from server " + ports[0]},
	}

	testUtils.AssertLoadBalancerResponse(t, testCases, port)
}

func TestRoundRobinRetryRequestTurnedOff(t *testing.T) {
	ports := testUtils.GetFreePorts(t, 3)
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Fatalf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, constants.RoundRobin)
	config.HealthCheckInterval = "30s"
	config.RetryRequests = false

	serverProcesses, _, port, teardownSuite := testUtils.SetupSuite(t, ports, config, nil)
	defer teardownSuite(t)

	testUtils.StopServer(serverProcesses[0])

	res, err := http.Get("http://localhost:" + strconv.Itoa(port))
	if err != nil {
		t.Fatalf("Error sending request to load balancer: %s", err)
	}
	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("Expected bad gateway status code, got %d", res.StatusCode)
	}
}

func TestRoundRobinRetryRequestTurnedOn(t *testing.T) {
	ports := testUtils.GetFreePorts(t, 3)
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Fatalf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, constants.RoundRobin)
	config.HealthCheckInterval = "30s"
	config.RetryRequests = true

	serverProcesses, _, port, teardownSuite := testUtils.SetupSuite(t, ports, config, nil)
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
