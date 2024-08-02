package loadbalancer

import (
	"errors"
	"hash/fnv"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

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
	case constants.LeastResponseTime:
		return func(w http.ResponseWriter, r *http.Request) {
			tlb.requestHandler(w, r, tlb.getNextServerLeastResponseTime)
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
	logger := slog.Default()
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

		// Make the request to the server
		proxy := server.GetReverseProxy()
		rec := httptest.NewRecorder()
		server.Mut.Lock()
		server.ActiveConnections++
		server.Mut.Unlock()
		start := time.Now()
		logger.Info("Sending request to server", slog.Attr{
			Key:   "Server",
			Value: slog.StringValue(server.URL.String()),
		}, slog.Attr{
			Key:   "Method",
			Value: slog.StringValue(r.Method),
		}, slog.Attr{
			Key:   "Path",
			Value: slog.StringValue(r.URL.Path),
		}, slog.Attr{
			Key:   "RemoteAddr",
			Value: slog.StringValue(r.RemoteAddr),
		})
		proxy.ServeHTTP(rec, r)
		elapsed := time.Since(start)

		// Update server statistics
		tlb.updateServerStats(server, elapsed)

		// If the response was OK, return the response, otherwise for loop continues and tries with the next server
		// This ensures fault tolerance and hides single server failures from the client
		if rec.Code < http.StatusInternalServerError {
			logger.Info("Sending response from server", slog.Attr{
				Key:   "Server",
				Value: slog.StringValue(server.URL.String()),
			}, slog.Attr{
				Key:   "duration",
				Value: slog.DurationValue(elapsed),
			}, slog.Attr{
				Key:   "status",
				Value: slog.IntValue(rec.Code),
			})
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

		logger.Info("Server", server.URL.String(), "returned status", slog.Attr{
			Key:   "status",
			Value: slog.IntValue(rec.Code),
		})
		tlb.setServerAsDead(server)
	}

	http.Error(w, "No healthy servers", http.StatusServiceUnavailable)
}

func (tlb *TinyLoadBalancer) updateServerStats(server *server.Server, elapsed time.Duration) {
	server.Mut.Lock()
	server.RequestsCount++
	server.RequestsDuration += elapsed
	server.Mut.Unlock()
}

func (tlb *TinyLoadBalancer) setServerAsDead(server *server.Server) {
	server.Mut.Lock()
	server.CurrentWeight = 0
	server.ActiveConnections = 0
	server.Healthy = false
	server.RequestsCount = 0
	server.RequestsDuration = 0
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

func (tlb *TinyLoadBalancer) getNextServerLeastResponseTime(_ string) (*server.Server, error) {
	tlb.Mut.Lock()
	defer tlb.Mut.Unlock()

	leastResponseTime := int64(math.MaxInt64)
	leastResponseTimeServer := -1
	for i := 0; i < len(tlb.Servers); i++ {
		if !tlb.Servers[i].Healthy {
			continue
		}
		if tlb.Servers[i].RequestsCount == 0 {
			return tlb.Servers[i], nil
		}

		avgResponseTime := int64(tlb.Servers[i].RequestsDuration) / tlb.Servers[i].RequestsCount
		if avgResponseTime < leastResponseTime {
			leastResponseTime = avgResponseTime
			leastResponseTimeServer = i
		}
	}

	if leastResponseTimeServer == -1 {
		return nil, errors.New("No healthy servers")
	}

	return tlb.Servers[leastResponseTimeServer], nil
}
