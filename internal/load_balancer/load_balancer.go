package loadbalancer

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/tiny-loadbalancer/internal/constants"
	"github.com/tiny-loadbalancer/internal/server"
)

type TinyLoadBalancer struct {
	Servers       []*server.Server
	Port          int
	Mut           sync.Mutex
	NextServer    int
	Strategy      constants.Strategy
	RetryRequests bool
}

func (tlb *TinyLoadBalancer) GetRequestHandler() http.HandlerFunc {
	switch tlb.Strategy {
	case constants.RoundRobin:
		return tlb.RoundRobinHandler

	default:
		return tlb.RoundRobinHandler
	}
}

func (tlb *TinyLoadBalancer) RoundRobinHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	for i := 0; i < len(tlb.Servers); i++ {
		var server *server.Server
		server, err = tlb.GetNextServerRoundRobin()
		if err != nil {
			http.Error(w, "No healthy servers", http.StatusServiceUnavailable)
			return
		}

		proxy := server.GetReverseProxy()
		// If we don't want to retry requests, just serve the request
		if !tlb.RetryRequests {
			proxy.ServeHTTP(w, r)
			return
		}

		rec := httptest.NewRecorder()
		proxy.ServeHTTP(rec, r)

		// If the server is healthy, return the response, otherwise try the next server
		if rec.Code < http.StatusInternalServerError {
			for k, v := range rec.Header() {
				w.Header()[k] = v
			}
			w.WriteHeader(rec.Code)
			io.Copy(w, rec.Body)
			return
		}

		fmt.Printf("Server %s returned status %d. Retrying with next server.\n", server.URL.String(), rec.Code)
		server.Mut.Lock()
		server.Healthy = false
		server.Mut.Unlock()
	}

	http.Error(w, "No healthy servers", http.StatusServiceUnavailable)
}

func (tlb *TinyLoadBalancer) GetNextServerRoundRobin() (*server.Server, error) {
	tlb.Mut.Lock()
	defer tlb.Mut.Unlock()

	server := tlb.Servers[tlb.NextServer]
	server.Mut.Lock()
	defer server.Mut.Unlock()
	if !server.Healthy {
		for i := 0; i < len(tlb.Servers); i++ {
			tlb.IncrementNextServer()
			server = tlb.Servers[tlb.NextServer]
			if server.Healthy {
				break
			}
		}
	}

	if !server.Healthy {
		return nil, errors.New("No healthy servers")
	}
	tlb.IncrementNextServer()

	return server, nil
}

func (tlb *TinyLoadBalancer) IncrementNextServer() {
	tlb.NextServer++
	if tlb.NextServer >= len(tlb.Servers) {
		tlb.NextServer = 0
	}
}
