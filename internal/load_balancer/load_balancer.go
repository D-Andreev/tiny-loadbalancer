package loadbalancer

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
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
		return func(w http.ResponseWriter, r *http.Request) {
			tlb.requestHandler(w, r, tlb.getNextServerRoundRobin)
		}
	case constants.Random:
		return func(w http.ResponseWriter, r *http.Request) {
			tlb.requestHandler(w, r, tlb.getNextServerRandom)
		}
	case constants.WeightedRoundRobin:
		return func(w http.ResponseWriter, r *http.Request) {
			tlb.requestHandler(w, r, tlb.getNextServerWeightedRoundRobin)
		}

	default:
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Strategy not supported", http.StatusBadRequest)
		}
	}
}

func (tlb *TinyLoadBalancer) requestHandler(
	w http.ResponseWriter,
	r *http.Request,
	getNextServer func() (*server.Server, error),
) {
	var err error
	tlb.Mut.Lock()
	shouldRetryRequests := tlb.RetryRequests
	serversCount := len(tlb.Servers)
	tlb.Mut.Unlock()

	for i := 0; i < serversCount; i++ {
		var server *server.Server
		server, err = getNextServer()
		if err != nil {
			http.Error(w, "No healthy servers", http.StatusServiceUnavailable)
			return
		}

		proxy := server.GetReverseProxy()
		// If we don't want to retry requests, just serve the request as is
		if !shouldRetryRequests {
			proxy.ServeHTTP(w, r)
			return
		}

		rec := httptest.NewRecorder()
		proxy.ServeHTTP(rec, r)

		// If the response was OK, return the response, otherwise for loop continues and tries with the next server
		// This ensures fault tolerance and hides single server failures from the client
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

func (tlb *TinyLoadBalancer) getNextServerRoundRobin() (*server.Server, error) {
	tlb.Mut.Lock()
	defer tlb.Mut.Unlock()

	server := tlb.Servers[tlb.NextServer]
	if !server.Healthy {
		for i := 0; i < len(tlb.Servers)-1; i++ {
			tlb.incrementNextServer()
			server = tlb.Servers[tlb.NextServer]
			if server.Healthy {
				break
			}
		}

		if !server.Healthy {
			return nil, errors.New("No healthy servers")
		}
	}

	tlb.incrementNextServer()

	return server, nil
}

func (tlb *TinyLoadBalancer) incrementNextServer() {
	tlb.NextServer++
	if tlb.NextServer >= len(tlb.Servers) {
		tlb.NextServer = 0
	}
}

func (tlb *TinyLoadBalancer) getNextServerRandom() (*server.Server, error) {
	tlb.Mut.Lock()
	defer tlb.Mut.Unlock()

	healthyServers := make([]*server.Server, 0)
	for _, s := range tlb.Servers {
		if s.Healthy {
			healthyServers = append(healthyServers, s)
		}
	}

	if len(healthyServers) == 0 {
		return nil, errors.New("No healthy servers")
	}

	max := len(healthyServers)
	idx := rand.Intn(max)

	return healthyServers[idx], nil
}

func (tlb *TinyLoadBalancer) getNextServerWeightedRoundRobin() (*server.Server, error) {
	tlb.Mut.Lock()
	defer tlb.Mut.Unlock()

	server := tlb.Servers[tlb.NextServer]
	if !server.Healthy || server.CurrentWeight == 0 {
		healthyServersCount := 0
		for i := 0; i < len(tlb.Servers)-1; i++ {
			tlb.incrementNextServer()
			server = tlb.Servers[tlb.NextServer]
			if server.Healthy {
				healthyServersCount++
			}
			if server.Healthy && server.CurrentWeight > 0 {
				break
			}
		}

		if healthyServersCount == 0 {
			return nil, errors.New("No healthy servers")
		}

		if server.CurrentWeight == 0 {
			tlb.resetServerWeights()
		}
	}

	server.CurrentWeight--
	tlb.incrementNextServer()

	return server, nil
}

func (tlb *TinyLoadBalancer) resetServerWeights() {
	for _, s := range tlb.Servers {
		s.CurrentWeight = s.Weight
	}
}
