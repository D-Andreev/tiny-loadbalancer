package e2e_tests

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	testUtils "github.com/tiny-loadbalancer/e2e_tests/test_utils"
)

func TestRoundRobin(t *testing.T) {
	ports := testUtils.GetFreePorts(t, 3)
	_, _, port, teardownSuite := testUtils.SetupSuite(t, ports)
	defer teardownSuite(t)

	testCases := []testUtils.TestCase{
		{Expected: "Hello from server " + ports[0]},
		{Expected: "Hello from server " + ports[1]},
		{Expected: "Hello from server " + ports[2]},
		{Expected: "Hello from server " + ports[0]},
		{Expected: "Hello from server " + ports[1]},
		{Expected: "Hello from server " + ports[2]},
		{Expected: "Hello from server " + ports[0]},
	}

	testUtils.AssertLoadBalancerResponse(t, testCases, port)
}

func TestNoServersAreStarted(t *testing.T) {
	_, _, port, teardownSuite := testUtils.SetupSuite(t, []string{})
	defer teardownSuite(t)

	_, err := http.Get("http://localhost:" + strconv.Itoa(port))
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestServerDiesAndComesBackOnline(t *testing.T) {
	ports := testUtils.GetFreePorts(t, 3)
	slaveProcesses, _, port, teardownSuite := testUtils.SetupSuite(t, ports)
	defer teardownSuite(t)

	testUtils.StopServer(slaveProcesses[0])
	time.Sleep(time.Second * 2)

	testCases := []testUtils.TestCase{
		{Expected: "Hello from server " + ports[1]},
		{Expected: "Hello from server " + ports[2]},
		{Expected: "Hello from server " + ports[1]},
		{Expected: "Hello from server " + ports[2]},
	}

	testUtils.AssertLoadBalancerResponse(t, testCases, port)

	cmd := testUtils.StartServer(ports[0])
	slaveProcesses[0] = cmd
	defer testUtils.StopServer(slaveProcesses[0])
	time.Sleep(time.Second * 2)

	testCases = []testUtils.TestCase{
		{Expected: "Hello from server " + ports[0]},
		{Expected: "Hello from server " + ports[1]},
		{Expected: "Hello from server " + ports[2]},
		{Expected: "Hello from server " + ports[0]},
		{Expected: "Hello from server " + ports[1]},
		{Expected: "Hello from server " + ports[2]},
		{Expected: "Hello from server " + ports[0]},
	}

	testUtils.AssertLoadBalancerResponse(t, testCases, port)
}
