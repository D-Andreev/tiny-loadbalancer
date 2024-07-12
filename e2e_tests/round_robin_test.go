package e2e_tests

import (
	"os/exec"
	"strings"
	"testing"
)

func TestRoundRobin(t *testing.T) {
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
