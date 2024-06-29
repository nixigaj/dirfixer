#!/bin/sh

env CGO_ENABLED=0 go build --ldflags="-w -s" -o dirfixer dirfixer.go
