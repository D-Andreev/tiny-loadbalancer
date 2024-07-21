package e2e_tests

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	testUtils "github.com/tiny-loadbalancer/e2e_tests/test_utils"
	"github.com/tiny-loadbalancer/internal/constants"
)

func TestLeastConnections(t *testing.T) {
	ports := testUtils.GetFreePorts(t, 3)
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Errorf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, constants.LeastConnections)
	_, _, port, teardownSuite := testUtils.SetupSuite(t, ports, config, nil)
	defer teardownSuite(t)

	testCases := []testUtils.TestCase{
		{ExpectedBody: "Hello from server " + ports[0], SlowResponse: true},
		{ExpectedBody: "Hello from server " + ports[1], SlowResponse: true},
		{ExpectedBody: "Hello from server " + ports[0], SlowResponse: true},
		{ExpectedBody: "Hello from server " + ports[1], SlowResponse: true},
		{ExpectedBody: "Hello from server " + ports[2], SlowResponse: true},
	}

	responses := testUtils.AssertLoadBalancerResponseAsync(t, testCases, port)

	for i, tc := range testCases {
		receivedResponse := false

		for _, res := range responses {
			if strings.TrimSpace(res) == tc.ExpectedBody {
				receivedResponse = true
				break
			}
		}

		if !receivedResponse {
			t.Errorf("Test case %d: Expected %s, got %s", i, tc.ExpectedBody, responses)
		}
	}
}

func TestLeastConnectionsNoServersAreStarted(t *testing.T) {
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Errorf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, constants.LeastConnections)
	_, _, port, teardownSuite := testUtils.SetupSuite(t, []string{}, config, nil)
	defer teardownSuite(t)

	res, _ := http.Get("http://localhost:" + strconv.Itoa(port))
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected service unavailable status code, got %d", res.StatusCode)
	}
}

func TestLeastConnectionsServerDiesAndComesBackOnline(t *testing.T) {
	ports := testUtils.GetFreePorts(t, 3)
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Errorf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, constants.LeastConnections)
	serverProcesses, _, port, teardownSuite := testUtils.SetupSuite(t, ports, config, nil)
	defer teardownSuite(t)

	testUtils.StopServer(serverProcesses[0])
	time.Sleep(time.Second * 2)

	testCases := []testUtils.TestCase{
		{ExpectedBody: "Hello from server " + ports[1]},
		{ExpectedBody: "Hello from server " + ports[1]},
	}

	testUtils.AssertLoadBalancerResponse(t, testCases, port)

	cmd := testUtils.StartServer(ports[0])
	serverProcesses[0] = cmd
	defer testUtils.StopServer(serverProcesses[0])
	time.Sleep(time.Second * 2)

	testCases = []testUtils.TestCase{
		{ExpectedBody: "Hello from server " + ports[0]},
		{ExpectedBody: "Hello from server " + ports[0]},
		{ExpectedBody: "Hello from server " + ports[0]},
	}

	testUtils.AssertLoadBalancerResponse(t, testCases, port)
}

func TestLeastConnectionsRetryRequestTurnedOff(t *testing.T) {
	ports := testUtils.GetFreePorts(t, 3)
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Errorf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, constants.LeastConnections)
	config.HealthCheckInterval = "30s"
	config.RetryRequests = false

	serverProcesses, _, port, teardownSuite := testUtils.SetupSuite(t, ports, config, nil)
	defer teardownSuite(t)

	testUtils.StopServer(serverProcesses[0])

	res, _ := http.Get("http://localhost:" + strconv.Itoa(port))
	if res.StatusCode != http.StatusBadGateway {
		t.Errorf("Expected bad gateway status code, got %d", res.StatusCode)
	}
}

func TestLeastConnectionsRetryRequestTurnedOn(t *testing.T) {
	ports := testUtils.GetFreePorts(t, 3)
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Errorf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port, constants.LeastConnections)
	config.HealthCheckInterval = "30s"
	config.RetryRequests = true

	serverProcesses, _, port, teardownSuite := testUtils.SetupSuite(t, ports, config, nil)
	defer teardownSuite(t)

	testUtils.StopServer(serverProcesses[0])

	testCases := []testUtils.TestCase{
		{ExpectedBody: "Hello from server " + ports[1]},
		{ExpectedBody: "Hello from server " + ports[1]},
		{ExpectedBody: "Hello from server " + ports[1]},
	}

	testUtils.AssertLoadBalancerResponse(t, testCases, port)
}
