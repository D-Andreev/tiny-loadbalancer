package e2e_tests

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"testing"
)

func TestRoundRobin(t *testing.T) {
	startServers()
	startLoadBalancer()

	testCases := []struct {
		expected string
	}{
		{"Hello from server 8081"},
		{"Hello from server 8082"},
		{"Hello from server 8083"},
		{"Hello from server 8081"},
		{"Hello from server 8082"},
		{"Hello from server 8083"},
		{"Hello from server 8081"},
	}

	for _, tc := range testCases {
		cmd := exec.Command("curl", "http://localhost:3333")
		output, _ := cmd.CombinedOutput()
		if !strings.Contains(string(output), tc.expected) {
			t.Errorf("Expected %s, got %s", tc.expected, output)
		}
	}
}

func startServers() {
	ports := []string{"8081", "8082", "8083"}
	for _, p := range ports {
		cmd := exec.Command("kill", fmt.Sprintf("$(lsof -t -i:%s)", p))
		err := cmd.Start()
		if err != nil {
			log.Fatal(err)
		}
		cmd = exec.Command("go", "run", "../servers/server.go", p)
		err = cmd.Start()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func startLoadBalancer() {
	cmd := exec.Command("kill", "$(lsof -t -i:3333)")
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	cmd = exec.Command("pwd")
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	output, _ := cmd.CombinedOutput()
	log.Println("PWD", string(output))
	cmd = exec.Command("go", "run", "../main.go", "../config.json")
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	output, _ = cmd.CombinedOutput()
	log.Println(string(output))
}
