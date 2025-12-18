#!/bin/bash
set -e
gofmt -s -w .
mkdir -p bin
GOOS=linux GOARCH=amd64 go build -o bin/gomon gomon.go
go build gomon.go
