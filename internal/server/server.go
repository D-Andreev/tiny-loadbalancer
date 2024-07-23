package server

import (
	"net/http/httputil"
	"net/url"
	"sync"
)

type Server struct {
	URL               *url.URL
	Mut               sync.Mutex
	Weight            int
	CurrentWeight     int
	ActiveConnections int
}

func NewServer(url *url.URL, weight int) *Server {
	return &Server{
		URL:               url,
		Weight:            weight,
		CurrentWeight:     weight,
		ActiveConnections: 0,
	}
}

func (s *Server) GetReverseProxy() *httputil.ReverseProxy {
	return httputil.NewSingleHostReverseProxy(s.URL)
}
