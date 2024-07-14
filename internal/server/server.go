package server

import (
	"net/http/httputil"
	"net/url"
	"sync"
)

type Server struct {
	URL     *url.URL
	Healthy bool
	Mut     sync.Mutex
}

func (s *Server) GetReverseProxy() *httputil.ReverseProxy {
	return httputil.NewSingleHostReverseProxy(s.URL)
}
