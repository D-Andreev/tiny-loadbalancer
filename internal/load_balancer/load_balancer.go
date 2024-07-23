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
	ServerPool    []*server.Server
	DeadServers   []*server.Server
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
	tlb.Mut.Unlock()

	for i := 0; i < len(tlb.ServerPool); i++ {
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
			tlb.SetServerAsDead(server)
			return
		}

		fmt.Printf("Server %s returned status %d. Retrying with next server.\n", server.URL.String(), rec.Code)
		tlb.SetServerAsDead(server)
	}

	http.Error(w, "No healthy servers", http.StatusBadGateway)
}

func (tlb *TinyLoadBalancer) SetServerAsDead(serverToKill *server.Server) {
	tlb.Mut.Lock()
	defer tlb.Mut.Unlock()
	serverToKill.CurrentWeight = 0
	serverToKill.ActiveConnections = 0

	var updatedServerPool []*server.Server
	for _, server := range tlb.ServerPool {
		if serverToKill == server {
			tlb.DeadServers = append(tlb.DeadServers, serverToKill)
			continue
		}
		updatedServerPool = append(updatedServerPool, server)
	}
	tlb.ServerPool = updatedServerPool
}

func (tlb *TinyLoadBalancer) SetServerAsAlive(s *server.Server) {
	tlb.Mut.Lock()
	defer tlb.Mut.Unlock()
	var updatedDeadServers []*server.Server
	for _, server := range tlb.DeadServers {
		if s == server {
			continue
		}
		updatedDeadServers = append(updatedDeadServers, server)
	}
	tlb.DeadServers = updatedDeadServers
	tlb.ServerPool = append(tlb.ServerPool, s)
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

	if len(tlb.ServerPool) == 0 {
		return nil, errors.New("No healthy servers")
	}

	server := tlb.ServerPool[tlb.NextServer]
	tlb.incrementNextServer()

	return server, nil
}

func (tlb *TinyLoadBalancer) incrementNextServer() {
	tlb.NextServer++
	if tlb.NextServer >= len(tlb.ServerPool) {
		tlb.NextServer = 0
	}
}

func (tlb *TinyLoadBalancer) getNextServerRandom(_ string) (*server.Server, error) {
	tlb.Mut.Lock()
	defer tlb.Mut.Unlock()

	if len(tlb.ServerPool) == 0 {
		return nil, errors.New("No healthy servers")
	}

	max := len(tlb.ServerPool)
	idx := rand.Intn(max)

	return tlb.ServerPool[idx], nil
}

func (tlb *TinyLoadBalancer) getNextServerWeightedRoundRobin(_ string) (*server.Server, error) {
	tlb.Mut.Lock()
	defer tlb.Mut.Unlock()

	if len(tlb.ServerPool) == 0 {
		return nil, errors.New("No healthy servers")
	}

	server := tlb.ServerPool[tlb.NextServer]
	if server.CurrentWeight == 0 {
		for range tlb.ServerPool {
			tlb.incrementNextServer()
			server = tlb.ServerPool[tlb.NextServer]
			if server.CurrentWeight > 0 {
				break
			}
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
	for _, s := range tlb.ServerPool {
		s.CurrentWeight = s.Weight
	}
}

func (tlb *TinyLoadBalancer) getNextServerIPHashing(ip string) (*server.Server, error) {
	tlb.Mut.Lock()
	defer tlb.Mut.Unlock()

	if len(tlb.ServerPool) == 0 {
		return nil, errors.New("No healthy servers")
	}

	hash := fnv.New32a()
	hash.Write([]byte(ip))
	hashedIP := hash.Sum32()

	idx := int(hashedIP) % len(tlb.ServerPool)
	server := tlb.ServerPool[idx]

	fmt.Println("IP Hashing: ", ip, " -> ", server.URL.String())

	return server, nil
}

func (tlb *TinyLoadBalancer) getNextServerLeastConnections(_ string) (*server.Server, error) {
	tlb.Mut.Lock()
	defer tlb.Mut.Unlock()

	if len(tlb.ServerPool) == 0 {
		return nil, errors.New("No healthy servers")
	}

	minActiveConnections := math.MaxInt32
	idx := -1
	for i, s := range tlb.ServerPool {
		if s.ActiveConnections < minActiveConnections {
			minActiveConnections = s.ActiveConnections
			idx = i
		}
	}

	return tlb.ServerPool[idx], nil
}
