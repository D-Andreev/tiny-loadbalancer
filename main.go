package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tiny-loadbalancer/internal/config"
	lb "github.com/tiny-loadbalancer/internal/load_balancer"
	"github.com/tiny-loadbalancer/internal/server"
)

func main() {
	logFile, err := initLogger()
	if err != nil {
		log.Fatalf("Failed to init log file %s", err)
	}
	defer logFile.Close()
	logger := slog.Default()

	args := os.Args[1:]
	if len(args) == 0 {
		logger.Error("Please provide a config file", "args", strings.Join(args, ", "))
		os.Exit(1)
	}
	configPath := args[0]
	c, err := initConfig(configPath)
	if err != nil {
		logger.Error("Error reading config file", "error", err)
		os.Exit(1)
	}

	healthCheckInterval, err := time.ParseDuration(c.HealthCheckInterval)
	if err != nil {
		logger.Error("Invalid health check interval", "error", err)
		os.Exit(1)
	}

	servers := getServers(c)
	tlb := &lb.TinyLoadBalancer{
		Port:          c.Port,
		Servers:       servers,
		Strategy:      c.Strategy,
		RetryRequests: c.RetryRequests,
	}

	// Run health checks for servers in interval
	for _, s := range tlb.Servers {
		go func(server *server.Server, logger *slog.Logger) {
			for range time.Tick(healthCheckInterval) {
				healthEndpointUrl := fmt.Sprintf("%s/health", server.URL.String())
				res, err := http.Get(healthEndpointUrl)
				if err != nil || res.StatusCode >= http.StatusInternalServerError {
					logger.Warn("Server is not healthy", slog.Attr{
						Key:   "Server",
						Value: slog.StringValue(server.URL.String()),
					})
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
		}(s, logger)
	}

	http.HandleFunc("/", tlb.GetRequestHandler())
	log.Println("Starting server on port", tlb.Port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", tlb.Port), nil)
	if err != nil {
		logger.Error("Error starting loadbalancer", "error", err)
		os.Exit(1)
	}
}

func initConfig(configPath string) (*config.Config, error) {
	config := &config.Config{}
	c, err := config.ReadConfig(configPath)
	err = config.ValidateConfig(c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func initLogger() (*os.File, error) {
	timestamp := time.Now().Unix()
	logPath := filepath.Join(".", "log")
	err := os.MkdirAll(logPath, os.ModePerm)
	if err != nil {
		return nil, err
	}
	logFileName := fmt.Sprintf(filepath.Join(logPath, "loadbalancer-%d.log"), timestamp)
	file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	mw := io.MultiWriter(os.Stdout, file)
	jsonHandler := slog.NewJSONHandler(mw, nil)
	slog.SetDefault(slog.New(jsonHandler))

	return file, nil
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
