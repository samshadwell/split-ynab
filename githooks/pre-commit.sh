#!/bin/sh

golangci-lint run ./...
go mod tidy && git add go.mod go.sum

