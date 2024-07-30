package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/tiny-loadbalancer/internal/config"
	lb "github.com/tiny-loadbalancer/internal/load_balancer"
	"github.com/tiny-loadbalancer/internal/server"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatal("Please provide a config file", args)
	}
	configPath := args[0]
	config, err := config.ReadConfig(configPath)

	healthCheckInterval, err := time.ParseDuration(config.HealthCheckInterval)
	if err != nil {
		log.Fatalf("Invalid health check interval: %s", err.Error())
	}

	servers := getServers(config)
	tlb := &lb.TinyLoadBalancer{
		Port:          config.Port,
		Servers:       servers,
		Strategy:      config.Strategy,
		RetryRequests: config.RetryRequests,
	}

	// Run health checks for servers in interval
	for _, s := range tlb.Servers {
		go func(server *server.Server) {
			for range time.Tick(healthCheckInterval) {
				healthEndpointUrl := fmt.Sprintf("%s/health", server.URL.String())
				res, err := http.Get(healthEndpointUrl)
				if err != nil || res.StatusCode >= http.StatusInternalServerError {
					fmt.Printf("Server %s is not healthy\n", healthEndpointUrl)
					server.Mut.Lock()
					server.Healthy = false
					server.Mut.Unlock()
				} else {
					defer res.Body.Close()
					server.Mut.Lock()
					server.Healthy = true
					server.Mut.Unlock()
				}
			}
		}(s)
	}

	http.HandleFunc("/", tlb.GetRequestHandler())
	log.Println("Starting server on port", tlb.Port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", tlb.Port), nil)
	if err != nil {
		panic(err)
	}
}

func getServers(config *config.Config) []*server.Server {
	var servers []*server.Server
	for _, s := range config.Servers {
		parsedUrl, err := url.Parse(s.Url)
		if err != nil {
			panic(err)
		}
		s := server.NewServer(parsedUrl, s.Weight)
		servers = append(servers, s)
	}

	return servers
}
