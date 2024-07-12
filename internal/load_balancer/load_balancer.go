package loadbalancer

import (
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
	server := tlb.GetNextServer()
	server.Proxy().ServeHTTP(w, r)
}

func (tlb *TinyLoadBalancer) GetNextServer() *server.Server {
	tlb.Mut.Lock()
	defer tlb.Mut.Unlock()

	server := tlb.Servers[tlb.NextServer]
	tlb.NextServer++
	if tlb.NextServer >= len(tlb.Servers) {
		tlb.NextServer = 0
	}
	return server
}
