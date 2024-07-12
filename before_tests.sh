#!/bin/bash

# Kill processes running on specified ports if they exist
for port in 3333 8081 8082 8083; do
  pid=$(lsof -t -i:$port)
  if [ -n "$pid" ]; then
    kill $pid
  fi
done

# Start the servers in the background
nohup go run ./servers/server.go 8081 > log/server1.log 2>&1 &
nohup go run ./servers/server.go 8082 > log/server2.log 2>&1 &
nohup go run ./servers/server.go 8083 > log/server3.log 2>&1 &

# Start the main application in the background
nohup go run ./main.go config.json > log/main.log 2>&1 &
