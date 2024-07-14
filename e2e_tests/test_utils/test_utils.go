package test_utils

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/tiny-loadbalancer/internal/config"
)

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

func StartLoadBalancer(port int, ports []string) *exec.Cmd {
	config := config.Config{
		Port:                strconv.Itoa(port),
		Strategy:            "round-robin",
		HealthCheckInterval: "1s",
	}
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

func StartServers(slaveProcesses []*exec.Cmd, ports []string) []*exec.Cmd {
	for _, port := range ports {
		cmd := exec.Command("go", "run", "servers/server.go", port)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := cmd.Start(); err != nil {
			log.Fatalf("Failed to start server on port %s: %v", port, err)
		}
		fmt.Printf("Started server on port: %s, with PID: %d\n", port, cmd.Process.Pid)
		slaveProcesses = append(slaveProcesses, cmd)
	}

	return slaveProcesses
}

func StopServers(slaveProcesses []*exec.Cmd) {
	for _, cmd := range slaveProcesses {
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
			log.Printf("Failed to kill server process: %v", err)
		}
		fmt.Printf("Stopped server with PID: %d\n", cmd.Process.Pid)
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
