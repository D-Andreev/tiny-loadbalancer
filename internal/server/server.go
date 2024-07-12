package server

import (
	"net/http/httputil"
	"net/url"
)

type Server struct {
	URL *url.URL
}

func (s *Server) Proxy() *httputil.ReverseProxy {
	return httputil.NewSingleHostReverseProxy(s.URL)
}
