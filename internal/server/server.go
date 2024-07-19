package server

import (
	"net/http/httputil"
	"net/url"
	"sync"
)

type Server struct {
	URL           *url.URL
	Healthy       bool
	Mut           sync.Mutex
	Weight        int
	CurrentWeight int
}

func NewServer(url *url.URL, weight int) *Server {
	return &Server{
		URL:           url,
		Healthy:       true,
		Weight:        weight,
		CurrentWeight: weight,
	}
}

func (s *Server) GetReverseProxy() *httputil.ReverseProxy {
	return httputil.NewSingleHostReverseProxy(s.URL)
}
