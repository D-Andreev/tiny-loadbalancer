package loadbalancer

import (
	"net/url"
	"testing"

	"github.com/tiny-loadbalancer/internal/server"
)

var ip = "127.0.0.1"

func TestRoundRobinGetNextServer(t *testing.T) {
	tlb := &TinyLoadBalancer{
		ServerPool: []*server.Server{
			{
				URL: &url.URL{Host: "localhost:8080"},
			},
			{
				URL: &url.URL{Host: "localhost:8081"},
			},
		},
		NextServer: 0,
	}

	server, err := tlb.getNextServerRoundRobin(ip)
	if err != nil {
		t.Fatalf("Error getting next server: %s", err.Error())
	}

	if server.URL.Host != "localhost:8080" {
		t.Fatalf("Expected server to be localhost:8080, got %s", server.URL.Host)
	}

	server, err = tlb.getNextServerRoundRobin(ip)
	if err != nil {
		t.Fatalf("Error getting next server: %s", err.Error())
	}

	if server.URL.Host != "localhost:8081" {
		t.Fatalf("Expected server to be localhost:8081, got %s", server.URL.Host)
	}

	server, err = tlb.getNextServerRoundRobin(ip)
	if err != nil {
		t.Fatalf("Error getting next server: %s", err.Error())
	}

	if server.URL.Host != "localhost:8080" {
		t.Fatalf("Expected server to be localhost:8080, got %s", server.URL.Host)
	}
}

func TestRoundRobinNextServerNoHealthyServers(t *testing.T) {
	tlb := &TinyLoadBalancer{
		ServerPool: []*server.Server{},
		NextServer: 0,
	}

	_, err := tlb.getNextServerRoundRobin(ip)
	if err == nil {
		t.Fatalf("Expected error getting next server")
	}
}

func TestRoundRobinNextServerOneUnhealthyServer(t *testing.T) {
	tlb := &TinyLoadBalancer{
		ServerPool: []*server.Server{
			{
				URL: &url.URL{Host: "localhost:8081"},
			},
		},
		NextServer: 0,
	}

	server, err := tlb.getNextServerRoundRobin(ip)
	if err != nil {
		t.Fatalf("Error getting next server: %s", err.Error())
	}
	if server.URL.Host != "localhost:8081" {
		t.Fatalf("Expected server to be localhost:8081, got %s", server.URL.Host)
	}
}

func TestWeightedRoundRobinNextserverNoHealthyServers(t *testing.T) {
	tlb := &TinyLoadBalancer{
		ServerPool: []*server.Server{},
		NextServer: 0,
	}

	_, err := tlb.getNextServerWeightedRoundRobin(ip)
	if err == nil {
		t.Fatalf("Expected error getting next server")
	}
}

func TestWeightedRoundRobinNextServer(t *testing.T) {
	tlb := &TinyLoadBalancer{
		ServerPool: []*server.Server{
			server.NewServer(&url.URL{Host: "localhost:8080"}, 5),
			server.NewServer(&url.URL{Host: "localhost:8081"}, 3),
			server.NewServer(&url.URL{Host: "localhost:8082"}, 2),
		},
		NextServer: 0,
	}

	testCases := []struct {
		expectedHost   string
		expectedWeight int
	}{
		// cycle one
		{expectedHost: "localhost:8080", expectedWeight: 4},
		{expectedHost: "localhost:8081", expectedWeight: 2},
		{expectedHost: "localhost:8082", expectedWeight: 1},
		{expectedHost: "localhost:8080", expectedWeight: 3},
		{expectedHost: "localhost:8081", expectedWeight: 1},
		{expectedHost: "localhost:8082", expectedWeight: 0},
		{expectedHost: "localhost:8080", expectedWeight: 2},
		{expectedHost: "localhost:8081", expectedWeight: 0},
		{expectedHost: "localhost:8080", expectedWeight: 1},
		{expectedHost: "localhost:8080", expectedWeight: 0},

		// cycle two
		{expectedHost: "localhost:8080", expectedWeight: 4},
		{expectedHost: "localhost:8081", expectedWeight: 2},
		{expectedHost: "localhost:8082", expectedWeight: 1},
		{expectedHost: "localhost:8080", expectedWeight: 3},
		{expectedHost: "localhost:8081", expectedWeight: 1},
		{expectedHost: "localhost:8082", expectedWeight: 0},
		{expectedHost: "localhost:8080", expectedWeight: 2},
		{expectedHost: "localhost:8081", expectedWeight: 0},
		{expectedHost: "localhost:8080", expectedWeight: 1},
		{expectedHost: "localhost:8080", expectedWeight: 0},
	}

	for i, tc := range testCases {
		server, err := tlb.getNextServerWeightedRoundRobin(ip)
		if err != nil {
			t.Fatalf("Error getting next server: %s", err.Error())
		}
		if server.URL.Host != tc.expectedHost {
			t.Fatalf("Test case %d: Expected server to be %s, got %s", i, tc.expectedHost, server.URL.Host)
		}
		if server.CurrentWeight != tc.expectedWeight {
			t.Fatalf("Test case %d: Expected weight to be %d, got %d", i, tc.expectedWeight, server.Weight)
		}
	}
}

func TestWeightedRoundRobinNextServerOneUnhealthyServer(t *testing.T) {
	tlb := &TinyLoadBalancer{
		ServerPool: []*server.Server{
			server.NewServer(&url.URL{Host: "localhost:8080"}, 5),
			server.NewServer(&url.URL{Host: "localhost:8081"}, 3),
			server.NewServer(&url.URL{Host: "localhost:8082"}, 2),
		},
		NextServer: 0,
	}
	tlb.SetServerAsDead(tlb.ServerPool[2])

	testCases := []struct {
		expectedHost   string
		expectedWeight int
	}{
		// cycle one
		{expectedHost: "localhost:8080", expectedWeight: 4},
		{expectedHost: "localhost:8081", expectedWeight: 2},
		{expectedHost: "localhost:8080", expectedWeight: 3},
		{expectedHost: "localhost:8081", expectedWeight: 1},
		{expectedHost: "localhost:8080", expectedWeight: 2},
		{expectedHost: "localhost:8081", expectedWeight: 0},
		{expectedHost: "localhost:8080", expectedWeight: 1},
		{expectedHost: "localhost:8080", expectedWeight: 0},

		// cycle two
		{expectedHost: "localhost:8080", expectedWeight: 4},
		{expectedHost: "localhost:8081", expectedWeight: 2},
		{expectedHost: "localhost:8080", expectedWeight: 3},
		{expectedHost: "localhost:8081", expectedWeight: 1},
		{expectedHost: "localhost:8080", expectedWeight: 2},
		{expectedHost: "localhost:8081", expectedWeight: 0},
		{expectedHost: "localhost:8080", expectedWeight: 1},
		{expectedHost: "localhost:8080", expectedWeight: 0},
	}

	for i, tc := range testCases {
		server, err := tlb.getNextServerWeightedRoundRobin(ip)
		if err != nil {
			t.Fatalf("Error getting next server: %s", err.Error())
		}
		if server.URL.Host != tc.expectedHost {
			t.Fatalf("Test case %d: Expected server to be %s, got %s", i, tc.expectedHost, server.URL.Host)
		}
		if server.CurrentWeight != tc.expectedWeight {
			t.Fatalf("Test case %d: Expected weight to be %d, got %d", i, tc.expectedWeight, server.Weight)
		}
	}
}

func TestWeightedRoundRobinNextServerFirstUnhealthyServer(t *testing.T) {
	tlb := &TinyLoadBalancer{
		ServerPool: []*server.Server{
			server.NewServer(&url.URL{Host: "localhost:8081"}, 3),
			server.NewServer(&url.URL{Host: "localhost:8082"}, 2),
		},
		NextServer: 0,
	}

	testCases := []struct {
		expectedHost   string
		expectedWeight int
	}{
		// cycle one
		{expectedHost: "localhost:8081", expectedWeight: 2},
		{expectedHost: "localhost:8082", expectedWeight: 1},
		{expectedHost: "localhost:8081", expectedWeight: 1},
		{expectedHost: "localhost:8082", expectedWeight: 0},
		{expectedHost: "localhost:8081", expectedWeight: 0},

		// cycle two
		{expectedHost: "localhost:8081", expectedWeight: 2},
		{expectedHost: "localhost:8082", expectedWeight: 1},
		{expectedHost: "localhost:8081", expectedWeight: 1},
		{expectedHost: "localhost:8082", expectedWeight: 0},
		{expectedHost: "localhost:8081", expectedWeight: 0},
	}

	for i, tc := range testCases {
		server, err := tlb.getNextServerWeightedRoundRobin(ip)
		if err != nil {
			t.Fatalf("Error getting next server: %s", err.Error())
		}
		if server.URL.Host != tc.expectedHost {
			t.Fatalf("Test case %d: Expected server to be %s, got %s", i, tc.expectedHost, server.URL.Host)
		}
		if server.CurrentWeight != tc.expectedWeight {
			t.Fatalf("Test case %d: Expected weight to be %d, got %d", i, tc.expectedWeight, server.Weight)
		}
	}
}

func TestIpHashingNextServer(t *testing.T) {
	tlb := &TinyLoadBalancer{
		ServerPool: []*server.Server{
			server.NewServer(&url.URL{Host: "localhost:8080"}, 0),
			server.NewServer(&url.URL{Host: "localhost:8081"}, 0),
			server.NewServer(&url.URL{Host: "localhost:8082"}, 0),
		},
		NextServer: 0,
	}

	testCases := []struct {
		ip           string
		expectedHost string
	}{
		{ip: "127.0.0.1", expectedHost: "localhost:8082"},
		{ip: "127.0.0.2", expectedHost: "localhost:8080"},
		{ip: "127.0.0.3", expectedHost: "localhost:8081"},
		{ip: "127.0.0.1", expectedHost: "localhost:8082"},
		{ip: "127.0.0.2", expectedHost: "localhost:8080"},
		{ip: "127.0.0.3", expectedHost: "localhost:8081"},
	}

	for i, tc := range testCases {
		server, err := tlb.getNextServerIPHashing(tc.ip)

		if err != nil {
			t.Fatalf("Error getting next server: %s", err.Error())
		}
		if server.URL.Host != tc.expectedHost {
			t.Fatalf("Test case %d: Expected server to be %s, got %s", i, tc.expectedHost, server.URL.Host)
		}
	}
}

func TestIpHashingNextServerUnhealthyServer(t *testing.T) {
	tlb := &TinyLoadBalancer{
		ServerPool: []*server.Server{
			server.NewServer(&url.URL{Host: "localhost:8080"}, 0),
			server.NewServer(&url.URL{Host: "localhost:8081"}, 0),
			server.NewServer(&url.URL{Host: "localhost:8082"}, 0),
		},
		NextServer: 0,
	}
	tlb.SetServerAsDead(tlb.ServerPool[0])

	testCases := []struct {
		ip           string
		expectedHost string
	}{
		{ip: "127.0.0.1", expectedHost: "localhost:8082"},
		{ip: "127.0.0.2", expectedHost: "localhost:8081"}, // 8080 is unhealthy, so it goes to next healthy server
		{ip: "127.0.0.3", expectedHost: "localhost:8081"},
		{ip: "127.0.0.1", expectedHost: "localhost:8082"},
		{ip: "127.0.0.2", expectedHost: "localhost:8081"}, // 8080 is unhealthy, so it goes to next healthy server
		{ip: "127.0.0.3", expectedHost: "localhost:8081"},
	}

	for i, tc := range testCases {
		server, err := tlb.getNextServerIPHashing(tc.ip)

		if err != nil {
			t.Fatalf("Error getting next server: %s", err.Error())
		}
		if server.URL.Host != tc.expectedHost {
			t.Fatalf("Test case %d: Expected server to be %s, got %s", i, tc.expectedHost, server.URL.Host)
		}
	}
}

func TestLeastConnections(t *testing.T) {
	tlb := &TinyLoadBalancer{
		ServerPool: []*server.Server{
			server.NewServer(&url.URL{Host: "localhost:8080"}, 0),
			server.NewServer(&url.URL{Host: "localhost:8081"}, 0),
			server.NewServer(&url.URL{Host: "localhost:8082"}, 0),
		},
		NextServer: 0,
	}
	tlb.ServerPool[0].ActiveConnections = 2
	tlb.ServerPool[1].ActiveConnections = 5
	tlb.ServerPool[2].ActiveConnections = 0

	testCases := []struct {
		expectedHost string
	}{
		{expectedHost: "localhost:8082"},
		{expectedHost: "localhost:8082"},
		{expectedHost: "localhost:8080"},
		{expectedHost: "localhost:8082"},
		{expectedHost: "localhost:8080"},
		{expectedHost: "localhost:8082"},
		{expectedHost: "localhost:8080"},
		{expectedHost: "localhost:8082"},
		{expectedHost: "localhost:8080"},
		{expectedHost: "localhost:8081"},
		{expectedHost: "localhost:8082"},
		{expectedHost: "localhost:8080"},
	}

	for i, tc := range testCases {
		server, err := tlb.getNextServerLeastConnections("")

		if err != nil {
			t.Fatalf("Error getting next server: %s", err.Error())
		}
		if server.URL.Host != tc.expectedHost {
			t.Fatalf("Test case %d: Expected server to be %s, got %s", i, tc.expectedHost, server.URL.Host)
		}
		server.ActiveConnections++
	}
}
func TestLeastConnectionsUnhealthyServer(t *testing.T) {
	tlb := &TinyLoadBalancer{
		ServerPool: []*server.Server{
			server.NewServer(&url.URL{Host: "localhost:8080"}, 0),
			server.NewServer(&url.URL{Host: "localhost:8081"}, 0),
			server.NewServer(&url.URL{Host: "localhost:8082"}, 0),
		},
		NextServer: 0,
	}
	tlb.ServerPool[0].ActiveConnections = 2
	tlb.ServerPool[1].ActiveConnections = 5
	tlb.ServerPool[2].ActiveConnections = 0
	tlb.SetServerAsDead(tlb.ServerPool[0])

	testCases := []struct {
		expectedHost string
	}{
		{expectedHost: "localhost:8082"},
		{expectedHost: "localhost:8082"},
		{expectedHost: "localhost:8082"},
		{expectedHost: "localhost:8082"},
		{expectedHost: "localhost:8082"},
		{expectedHost: "localhost:8081"},
		{expectedHost: "localhost:8082"},
	}

	for i, tc := range testCases {
		server, err := tlb.getNextServerLeastConnections("")

		if err != nil {
			t.Fatalf("Error getting next server: %s", err.Error())
		}
		if server.URL.Host != tc.expectedHost {
			t.Fatalf("Test case %d: Expected server to be %s, got %s", i, tc.expectedHost, server.URL.Host)
		}
		server.ActiveConnections++
	}
}
