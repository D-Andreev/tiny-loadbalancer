package loadbalancer

import (
	"errors"
	"net/http"
	"sync"

	"github.com/tiny-loadbalancer/internal/constants"
	"github.com/tiny-loadbalancer/internal/server"
)

type TinyLoadBalancer struct {
	Servers    []*server.Server
	Port       string
	Mut        sync.Mutex
	NextServer int
	Strategy   constants.Strategy
}

func (tlb *TinyLoadBalancer) HandleRequest(w http.ResponseWriter, r *http.Request) {
	var server *server.Server
	var err error
	switch tlb.Strategy {
	case constants.RoundRobin:
		server, err = tlb.GetNextServerRoundRobin()
	default:
		server, err = tlb.GetNextServerRoundRobin()
	}
	if err != nil {
		http.Error(w, "No healthy servers", http.StatusServiceUnavailable)
		return
	}
	server.Proxy().ServeHTTP(w, r)
}

func (tlb *TinyLoadBalancer) GetNextServerRoundRobin() (*server.Server, error) {
	tlb.Mut.Lock()
	defer tlb.Mut.Unlock()

	server := tlb.Servers[tlb.NextServer]
	server.Mut.Lock()
	defer server.Mut.Unlock()
	if !server.Healthy {
		for i := 0; i < len(tlb.Servers); i++ {
			tlb.NextServer++
			if tlb.NextServer >= len(tlb.Servers) {
				tlb.NextServer = 0
			}
			server = tlb.Servers[tlb.NextServer]
			if server.Healthy {
				break
			}
		}
	}

	if !server.Healthy {
		return nil, errors.New("No healthy servers")
	}

	tlb.NextServer++
	if tlb.NextServer >= len(tlb.Servers) {
		tlb.NextServer = 0
	}
	return server, nil
}
