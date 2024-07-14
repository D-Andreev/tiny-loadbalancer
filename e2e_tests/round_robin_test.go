package e2e_tests

import (
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	testUtils "github.com/tiny-loadbalancer/e2e_tests/test_utils"
)

func TestRoundRobin(t *testing.T) {
	ports := []string{}
	for i := 0; i < 3; i++ {
		port, err := testUtils.GetFreePort()
		if err != nil {
			log.Fatalf("Failed to get free port: %v", err)
		}
		ports = append(ports, strconv.Itoa(port))
	}
	_, _, port, teardownSuite := testUtils.SetupSuite(t, ports)
	defer teardownSuite(t)

	testCases := []struct {
		expected string
	}{
		{"Hello from server " + ports[0]},
		{"Hello from server " + ports[1]},
		{"Hello from server " + ports[2]},
		{"Hello from server " + ports[0]},
		{"Hello from server " + ports[1]},
		{"Hello from server " + ports[2]},
		{"Hello from server " + ports[0]},
	}

	assertResponse(t, testCases, port)
}

func TestWithUnhealthyServer(t *testing.T) {
	ports := []string{}
	for i := 0; i < 2; i++ {
		port, err := testUtils.GetFreePort()
		if err != nil {
			log.Fatalf("Failed to get free port: %v", err)
		}
		ports = append(ports, strconv.Itoa(port))
	}
	_, _, port, teardownSuite := testUtils.SetupSuite(t, ports)
	defer teardownSuite(t)

	testCases := []struct {
		expected string
	}{
		{"Hello from server " + ports[0]},
		{"Hello from server " + ports[1]},
		{"Hello from server " + ports[0]},
		{"Hello from server " + ports[1]},
		{"Hello from server " + ports[0]},
	}

	assertResponse(t, testCases, port)
}

func assertResponse(t *testing.T, testCases []struct{ expected string }, port int) {
	t.Helper()
	for _, tc := range testCases {
		res, err := http.Get("http://localhost:" + strconv.Itoa(port))
		if err != nil {
			t.Errorf("Error making request: %s", err.Error())
		}
		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			t.Errorf("Error reading response body: %s", err.Error())
		}
		defer res.Body.Close()
		if !strings.Contains(string(resBody), tc.expected) {
			t.Errorf("Expected %s, got %s", tc.expected, resBody)
		}
	}
}

func TestWithNoHealthyServers(t *testing.T) {
	_, _, port, teardownSuite := testUtils.SetupSuite(t, []string{})
	defer teardownSuite(t)

	_, err := http.Get("http://localhost:" + strconv.Itoa(port))
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestServerComesBackOnline(t *testing.T) {
	ports := []string{}
	for i := 0; i < 3; i++ {
		port, err := testUtils.GetFreePort()
		if err != nil {
			log.Fatalf("Failed to get free port: %v", err)
		}
		ports = append(ports, strconv.Itoa(port))
	}
	slaveProcesses, _, port, teardownSuite := testUtils.SetupSuite(t, ports)
	defer teardownSuite(t)

	testUtils.StopServer(slaveProcesses[0])
	time.Sleep(time.Second * 2)

	testCases := []struct {
		expected string
	}{
		{"Hello from server " + ports[1]},
		{"Hello from server " + ports[2]},
		{"Hello from server " + ports[1]},
		{"Hello from server " + ports[2]},
	}

	assertResponse(t, testCases, port)

	cmd := testUtils.StartServer(ports[0])
	slaveProcesses[0] = cmd
	defer testUtils.StopServer(slaveProcesses[0])
	time.Sleep(time.Second * 2)

	testCases = []struct {
		expected string
	}{
		{"Hello from server " + ports[0]},
		{"Hello from server " + ports[1]},
		{"Hello from server " + ports[2]},
		{"Hello from server " + ports[0]},
		{"Hello from server " + ports[1]},
		{"Hello from server " + ports[2]},
		{"Hello from server " + ports[0]},
	}

	assertResponse(t, testCases, port)
}
