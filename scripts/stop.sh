#!/bin/bash
pkill -f "go run cmd/server/main.go"
kill -9 $(lsof -ti:8081) 2>/dev/null || echo "端口8081未被占用"
