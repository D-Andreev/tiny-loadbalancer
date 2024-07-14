package e2e_tests

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	testUtils "github.com/tiny-loadbalancer/e2e_tests/test_utils"
)

func TestRandom(t *testing.T) {
	ports := testUtils.GetFreePorts(t, 3)
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Errorf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port)
	config.Strategy = "random"
	_, _, port, teardownSuite := testUtils.SetupSuite(t, ports, config)
	defer teardownSuite(t)

	testCases := []testUtils.TestCase{
		{ExpectedStatusCode: 200},
		{ExpectedStatusCode: 200},
		{ExpectedStatusCode: 200},
		{ExpectedStatusCode: 200},
	}

	testUtils.AssertLoadBalancerStatusCode(t, testCases, port)
}

func TestRandomNoServersAreStarted(t *testing.T) {
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Errorf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port)
	config.Strategy = "random"
	_, _, port, teardownSuite := testUtils.SetupSuite(t, []string{}, config)
	defer teardownSuite(t)

	res, _ := http.Get("http://localhost:" + strconv.Itoa(port))
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected service unavailable status code, got %d", res.StatusCode)
	}
}

func TestRandomServerDiesAndComesBackOnline(t *testing.T) {
	ports := testUtils.GetFreePorts(t, 2)
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Errorf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port)
	config.Strategy = "random"
	serverProcesses, _, port, teardownSuite := testUtils.SetupSuite(t, ports, config)
	defer teardownSuite(t)

	testUtils.StopServer(serverProcesses[0])
	time.Sleep(time.Second * 2)

	testCases := []testUtils.TestCase{
		{ExpectedBody: "Hello from server " + ports[1]},
		{ExpectedBody: "Hello from server " + ports[1]},
		{ExpectedBody: "Hello from server " + ports[1]},
	}

	testUtils.AssertLoadBalancerResponse(t, testCases, port)

	testUtils.StopServer(serverProcesses[1])
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

func TestRandomRetryRequestTurnedOff(t *testing.T) {
	ports := testUtils.GetFreePorts(t, 1)
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Errorf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port)
	config.Strategy = "random"
	config.HealthCheckInterval = "30s"
	config.RetryRequests = false

	serverProcesses, _, port, teardownSuite := testUtils.SetupSuite(t, ports, config)
	defer teardownSuite(t)

	testUtils.StopServer(serverProcesses[0])

	res, _ := http.Get("http://localhost:" + strconv.Itoa(port))
	if res.StatusCode != http.StatusBadGateway {
		t.Errorf("Expected bad gateway status code, got %d", res.StatusCode)
	}
}

func TestRandomRetryRequestTurnedOn(t *testing.T) {
	ports := testUtils.GetFreePorts(t, 2)
	port, err := testUtils.GetFreePort()
	if err != nil {
		t.Errorf("Error getting free port for load balancer")
	}
	config := testUtils.GetConfig(port)
	config.Strategy = "random"
	config.HealthCheckInterval = "30s"
	config.RetryRequests = true

	serverProcesses, _, port, teardownSuite := testUtils.SetupSuite(t, ports, config)
	defer teardownSuite(t)

	testUtils.StopServer(serverProcesses[0])

	testCases := make([]testUtils.TestCase, 0)
	/*
		Because the next server is picked randomly, we are going to spam the load balancer with requests
		to make sure that the stopped server (Server 1) is picked at least once and the request is retried
		100 requests should be plenty.
	*/
	for i := 0; i < 100; i++ {
		testCases = append(testCases, testUtils.TestCase{ExpectedBody: "Hello from server " + ports[1]})
	}

	testUtils.AssertLoadBalancerResponse(t, testCases, port)
}
