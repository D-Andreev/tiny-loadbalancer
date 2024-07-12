#!/bin/bash

# Kill processes running on specified ports if they exist
for port in 3333 8081 8082 8083; do
  pid=$(lsof -t -i:$port)
  if [ -n "$pid" ]; then
    kill $pid
  fi
done
