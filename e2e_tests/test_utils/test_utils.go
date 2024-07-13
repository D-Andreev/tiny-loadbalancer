package test_utils

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
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
		HealthCheckInterval: "5s",
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
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start load balancer: %v", err)
	}
	return cmd
}

func StopLoadBalancer(loadBalancerProcess *exec.Cmd) {
	if err := loadBalancerProcess.Process.Release(); err != nil {
		log.Printf("Failed to kill load balancer process: %v", err)
	}
	e := os.Remove("../config-test.json")
	if e != nil {
		log.Fatal(e)
	}
}

func StartSlave(port string) *exec.Cmd {
	cmd := exec.Command("go", "run", "servers/server.go", port)
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start slave on port %s: %v", port, err)
	}
	return cmd
}

func StartSlaves(slaveProcesses []*exec.Cmd, ports []string) []*exec.Cmd {
	for _, port := range ports {
		cmd := StartSlave(port)
		slaveProcesses = append(slaveProcesses, cmd)
	}

	return slaveProcesses
}

func StopSlaves(slaveProcesses []*exec.Cmd) {
	for _, cmd := range slaveProcesses {
		if err := cmd.Process.Release(); err != nil {
			log.Printf("Failed to kill slave process: %v", err)
		}
	}
}

func SetupSuite(_ *testing.T, ports []string) (int, func(t *testing.T)) {
	var slaveProcesses []*exec.Cmd
	var loadBalancerProcess *exec.Cmd
	slaveProcesses = StartSlaves(slaveProcesses, ports)
	port, err := GetFreePort()
	if err != nil {
		log.Fatalf("Failed to get free port: %v", err)
	}
	loadBalancerProcess = StartLoadBalancer(port, ports)

	time.Sleep(5 * time.Second)

	return port, func(t *testing.T) {
		StopSlaves(slaveProcesses)
		StopLoadBalancer(loadBalancerProcess)
	}
}
