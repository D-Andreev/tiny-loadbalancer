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
	for _, s := range servers {
		go func(ss *server.Server) {
			for range time.Tick(healthCheckInterval) {
				healthEndpointUrl := fmt.Sprintf("%s/health", ss.URL.String())
				res, err := http.Get(healthEndpointUrl)
				if err != nil || res.StatusCode >= 500 {
					fmt.Printf("Server %s is not healthy\n", healthEndpointUrl)
					ss.Healthy = false
				} else {
					defer res.Body.Close()
					ss.Healthy = true
				}
			}
		}(s)
	}

	tlb := &lb.TinyLoadBalancer{
		Port:     config.Port,
		Servers:  servers,
		Strategy: config.Strategy,
	}

	http.HandleFunc("/", tlb.HandleRequest)
	log.Println("Starting server on port", tlb.Port)
	err = http.ListenAndServe(fmt.Sprintf(":%s", tlb.Port), nil)
	if err != nil {
		panic(err)
	}
}

func getServers(config *config.Config) []*server.Server {
	var servers []*server.Server
	for _, serverUrl := range config.ServerUrls {
		parsedUrl, err := url.Parse(serverUrl)
		if err != nil {
			panic(err)
		}
		s := &server.Server{
			URL:     parsedUrl,
			Healthy: true,
		}
		servers = append(servers, s)
	}

	return servers
}
