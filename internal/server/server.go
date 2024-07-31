package server

import (
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

type Server struct {
	URL               *url.URL
	Healthy           bool
	Mut               sync.Mutex
	Weight            int
	CurrentWeight     int
	ActiveConnections int
	RequestsCount     int64
	RequestsDuration  time.Duration
}

func NewServer(url *url.URL, weight int) *Server {
	return &Server{
		URL:               url,
		Healthy:           true,
		Weight:            weight,
		CurrentWeight:     weight,
		ActiveConnections: 0,
		RequestsCount:     0,
		RequestsDuration:  0,
	}
}

func (s *Server) GetReverseProxy() *httputil.ReverseProxy {
	return httputil.NewSingleHostReverseProxy(s.URL)
}
