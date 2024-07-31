package e2e_tests

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	testUtils "github.com/tiny-loadbalancer/e2e_tests/test_utils"
	"github.com/tiny-loadbalancer/internal/constants"
)

func TestLeastResponseTime(t *testing.T) {
	ports := testUtils.GetFreePorts(t, 3)
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Fatalf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, constants.LeastResponseTime)
	_, _, port, teardownSuite := testUtils.SetupSuite(t, ports, config, nil)
	defer teardownSuite(t)

	testCases := []testUtils.TestCase{
		{ExpectedBody: "Hello from server " + ports[0], SlowResponse: true, Duration: 100},
		{ExpectedBody: "Hello from server " + ports[1], SlowResponse: true, Duration: 50},
		{ExpectedBody: "Hello from server " + ports[2], SlowResponse: true, Duration: 200},
		{ExpectedBody: "Hello from server " + ports[1], SlowResponse: true, Duration: 100},
		{ExpectedBody: "Hello from server " + ports[1], SlowResponse: true, Duration: 600},
		{ExpectedBody: "Hello from server " + ports[0], SlowResponse: true, Duration: 700},
		{ExpectedBody: "Hello from server " + ports[2], SlowResponse: true, Duration: 100},
		{ExpectedBody: "Hello from server " + ports[2], SlowResponse: true, Duration: 1500},
		{ExpectedBody: "Hello from server " + ports[1], SlowResponse: true, Duration: 1500},
		{ExpectedBody: "Hello from server " + ports[0], SlowResponse: true, Duration: 900},
	}

	testUtils.AssertLoadBalancerResponse(t, testCases, port)
}

func TestLeastResponseTimeNoHealthyServers(t *testing.T) {
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Fatalf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, constants.LeastResponseTime)
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

func TestLeastResponseTimeServerDiesAndComesBackOnline(t *testing.T) {
	ports := testUtils.GetFreePorts(t, 3)
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Fatalf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, constants.LeastResponseTime)
	serverProcesses, _, port, teardownSuite := testUtils.SetupSuite(t, ports, config, nil)
	defer teardownSuite(t)

	testUtils.StopServer(serverProcesses[0])
	time.Sleep(time.Second * 2)

	testCases := []testUtils.TestCase{
		{ExpectedBody: "Hello from server " + ports[1], SlowResponse: true},
		{ExpectedBody: "Hello from server " + ports[2]},
	}

	testUtils.AssertLoadBalancerResponseAsync(t, testCases, port)

	cmd := testUtils.StartServer(ports[0])
	serverProcesses[0] = cmd
	defer testUtils.StopServer(serverProcesses[0])
	time.Sleep(time.Second * 2)

	testCases = []testUtils.TestCase{
		{ExpectedBody: "Hello from server " + ports[0], SlowResponse: true},
		{ExpectedBody: "Hello from server " + ports[1], SlowResponse: true},
		{ExpectedBody: "Hello from server " + ports[2], SlowResponse: true},
	}

	testUtils.AssertLoadBalancerResponseAsync(t, testCases, port)
}

func TestLeastResponseTimeRetryRequestTurnedOff(t *testing.T) {
	ports := testUtils.GetFreePorts(t, 3)
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Fatalf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, constants.LeastResponseTime)
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

func TestLeastResponseTimeRetryRequestTurnedOn(t *testing.T) {
	ports := testUtils.GetFreePorts(t, 3)
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Fatalf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, constants.LeastResponseTime)
	config.HealthCheckInterval = "30s"
	config.RetryRequests = true

	serverProcesses, _, port, teardownSuite := testUtils.SetupSuite(t, ports, config, nil)
	defer teardownSuite(t)

	testUtils.StopServer(serverProcesses[0])

	testCases := []testUtils.TestCase{
		{ExpectedBody: "Hello from server " + ports[1], SlowResponse: true, Duration: 100},
		{ExpectedBody: "Hello from server " + ports[2], SlowResponse: true, Duration: 200},
		{ExpectedBody: "Hello from server " + ports[1]},
	}

	testUtils.AssertLoadBalancerResponse(t, testCases, port)
}
