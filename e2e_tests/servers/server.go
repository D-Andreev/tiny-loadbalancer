package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

/*
	This is a simple server that listens on a port and returns a message with the port number.
	To run this server, you need to pass the port number as an argument. For example:
		go run server.go 8081
	We're starting servers before running the load balancer, so we can test the load balancer.
*/

func main() {
	args := os.Args[1:]
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/slow" {
			// Simulate work
			duration := 1000
			var err error
			if r.URL.Query().Get("duration") != "" {
				duration, err = strconv.Atoi(r.URL.Query().Get("duration"))
				if err != nil {
					http.Error(w, "Invalid timeout", http.StatusBadRequest)
					return
				}
			}
			time.Sleep(time.Duration(duration) * time.Millisecond)
			w.Write([]byte(fmt.Sprintf("Hello from server %s\n", args[0])))
			return
		}

		w.Write([]byte(fmt.Sprintf("Hello from server %s\n", args[0])))
	})

	port := fmt.Sprintf(":%s", args[0])
	fmt.Println("Starting server on port", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		panic(err)
	}
}
