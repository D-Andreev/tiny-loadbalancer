package e2e_tests

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
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
	t.Skip()
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
	if err := syscall.Kill(-slaveProcesses[0].Process.Pid, syscall.SIGKILL); err != nil {
		t.Errorf("Error releasing process: %s", err.Error())
	}
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

	cmd := exec.Command("go", "run", "servers/server.go", ports[0])
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start server on port %s: %v", ports[0], err)
	}
	time.Sleep(time.Second * 3)
	fmt.Println("Ports:", ports)
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
