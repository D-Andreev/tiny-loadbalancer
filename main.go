package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

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

	servers := initServers(config)
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

func initServers(config *config.Config) []*server.Server {
	var servers []*server.Server
	for _, serverUrl := range config.ServerUrls {
		parsedUrl, err := url.Parse(serverUrl)
		if err != nil {
			panic(err)
		}
		s := &server.Server{
			URL: parsedUrl,
		}
		servers = append(servers, s)
	}

	return servers
}
