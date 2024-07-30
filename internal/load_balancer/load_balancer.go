package loadbalancer

import (
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"math"
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
	case constants.IPHashing:
		return func(w http.ResponseWriter, r *http.Request) {
			tlb.requestHandler(w, r, tlb.getNextServerIPHashing)
		}
	case constants.LeastConnections:
		return func(w http.ResponseWriter, r *http.Request) {
			tlb.requestHandler(w, r, tlb.getNextServerLeastConnections)
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
	getNextServer func(ip string) (*server.Server, error),
) {
	var err error
	tlb.Mut.Lock()
	shouldRetryRequests := tlb.RetryRequests
	serversCount := len(tlb.Servers)
	tlb.Mut.Unlock()

	for i := 0; i < serversCount; i++ {
		var server *server.Server
		server, err = getNextServer(r.RemoteAddr)
		if err != nil {
			http.Error(w, "No healthy servers", http.StatusServiceUnavailable)
			return
		}

		proxy := server.GetReverseProxy()
		rec := httptest.NewRecorder()
		server.Mut.Lock()
		server.ActiveConnections++
		server.Mut.Unlock()
		proxy.ServeHTTP(rec, r)

		// If the response was OK, return the response, otherwise for loop continues and tries with the next server
		// This ensures fault tolerance and hides single server failures from the client
		if rec.Code < http.StatusInternalServerError {
			tlb.returnResponse(rec, w)
			server.Mut.Lock()
			server.ActiveConnections--
			server.Mut.Unlock()
			return
		}

		// If we don't want to retry requests, just return the response
		if !shouldRetryRequests {
			tlb.returnResponse(rec, w)
			tlb.setServerAsDead(server)
			return
		}

		fmt.Printf("Server %s returned status %d. Retrying with next server.\n", server.URL.String(), rec.Code)
		tlb.setServerAsDead(server)
	}

	http.Error(w, "No healthy servers", http.StatusServiceUnavailable)
}

func (tlb *TinyLoadBalancer) setServerAsDead(server *server.Server) {
	server.Mut.Lock()
	server.CurrentWeight = 0
	server.ActiveConnections = 0
	server.Healthy = false
	server.Mut.Unlock()
}

func (tlb *TinyLoadBalancer) returnResponse(rec *httptest.ResponseRecorder, w http.ResponseWriter) {
	for k, v := range rec.Header() {
		w.Header()[k] = v
	}
	w.WriteHeader(rec.Code)
	io.Copy(w, rec.Body)
}

func (tlb *TinyLoadBalancer) getNextServerRoundRobin(_ string) (*server.Server, error) {
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

func (tlb *TinyLoadBalancer) getNextServerRandom(_ string) (*server.Server, error) {
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

func (tlb *TinyLoadBalancer) getNextServerWeightedRoundRobin(_ string) (*server.Server, error) {
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

func (tlb *TinyLoadBalancer) getNextServerIPHashing(ip string) (*server.Server, error) {
	tlb.Mut.Lock()
	defer tlb.Mut.Unlock()

	hash := fnv.New32a()
	hash.Write([]byte(ip))
	hashedIP := hash.Sum32()

	idx := int(hashedIP) % len(tlb.Servers)
	server := tlb.Servers[idx]

	if !server.Healthy {
		for i := 0; i < len(tlb.Servers)-1; i++ {
			idx++
			if idx >= len(tlb.Servers) {
				idx = 0
			}
			server = tlb.Servers[idx]
			if server.Healthy {
				break
			}
		}

		if !server.Healthy {
			return nil, errors.New("No healthy servers")
		}
	}

	return server, nil
}

func (tlb *TinyLoadBalancer) getNextServerLeastConnections(_ string) (*server.Server, error) {
	tlb.Mut.Lock()
	defer tlb.Mut.Unlock()

	minActiveConnections := math.MaxInt32
	idx := -1
	for i := 0; i < len(tlb.Servers); i++ {
		if tlb.Servers[i].ActiveConnections < minActiveConnections && tlb.Servers[i].Healthy {
			minActiveConnections = tlb.Servers[i].ActiveConnections
			idx = i
		}
	}
	if idx == -1 {
		return nil, errors.New("No healthy servers")
	}

	return tlb.Servers[idx], nil
}
