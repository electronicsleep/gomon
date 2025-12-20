#!/bin/bash
set -e
gofmt -s -w .
mkdir -p bin
GOOS=linux GOARCH=amd64 go build -o bin/gomon_amd64 gomon.go
GOOS=linux GOARCH=arm go build -o bin/gomon_arm gomon.go
go build gomon.go
