package test_utils

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/tiny-loadbalancer/internal/config"
)

type TestCase struct {
	Expected string
}

func GetFreePort() (port int, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return
}

func WriteConfigFile(config config.Config, ports []string) {
	for _, p := range ports {
		config.ServerUrls = append(config.ServerUrls, "http://localhost:"+p)
	}
	content, err := json.Marshal(config)
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile("../config-test.json", content, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func StartLoadBalancer(port int, ports []string) *exec.Cmd {
	config := config.Config{
		Port:                strconv.Itoa(port),
		Strategy:            "round-robin",
		HealthCheckInterval: "1s",
	}
	WriteConfigFile(config, ports)

	cmd := exec.Command("go", "run", "../main.go", "../config-test.json")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start load balancer: %v", err)
	}
	fmt.Printf("Started load balancer on port %d, with PID: %d\n", port, cmd.Process.Pid)

	return cmd
}

func StopLoadBalancer(loadBalancerProcess *exec.Cmd) {
	if err := syscall.Kill(-loadBalancerProcess.Process.Pid, syscall.SIGKILL); err != nil {
		log.Printf("Failed to kill load balancer process: %v", err)
	}
	fmt.Printf("Stopped load balancer with PID: %d\n", loadBalancerProcess.Process.Pid)
	e := os.Remove("../config-test.json")
	if e != nil {
		log.Fatal(e)
	}
}

func StartServer(port string) *exec.Cmd {
	cmd := exec.Command("go", "run", "servers/server.go", port)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start server on port %s: %v", port, err)
	}
	fmt.Printf("Started server on port: %s, with PID: %d\n", port, cmd.Process.Pid)

	return cmd
}

func StartServers(slaveProcesses []*exec.Cmd, ports []string) []*exec.Cmd {
	for _, port := range ports {
		cmd := StartServer(port)
		slaveProcesses = append(slaveProcesses, cmd)
	}

	return slaveProcesses
}

func StopServer(cmd *exec.Cmd) {
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
		log.Printf("Failed to kill server process: %v", err)
	}
	fmt.Printf("Stopped server with PID: %d\n", cmd.Process.Pid)
}

func StopServers(slaveProcesses []*exec.Cmd) {
	for _, cmd := range slaveProcesses {
		StopServer(cmd)
	}
}

func SetupSuite(_ *testing.T, ports []string) ([]*exec.Cmd, *exec.Cmd, int, func(t *testing.T)) {
	var slaveProcesses []*exec.Cmd
	var loadBalancerProcess *exec.Cmd
	slaveProcesses = StartServers(slaveProcesses, ports)
	port, err := GetFreePort()
	if err != nil {
		log.Fatalf("Failed to get free port: %v", err)
	}
	loadBalancerProcess = StartLoadBalancer(port, ports)

	time.Sleep(2 * time.Second)

	return slaveProcesses, loadBalancerProcess, port, func(t *testing.T) {
		StopServers(slaveProcesses)
		StopLoadBalancer(loadBalancerProcess)
	}
}

func GetFreePorts(t *testing.T, n int) []string {
	t.Helper()
	ports := []string{}
	for i := 0; i < n; i++ {
		port, err := GetFreePort()
		if err != nil {
			t.Errorf(err.Error())
		}
		ports = append(ports, strconv.Itoa(port))
	}

	return ports
}

func AssertLoadBalancerResponse(t *testing.T, testCases []TestCase, port int) {
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
		if !strings.Contains(string(resBody), tc.Expected) {
			t.Errorf("Expected %s, got %s", tc.Expected, resBody)
		}
	}
}
