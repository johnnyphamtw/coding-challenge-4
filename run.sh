#!/bin/bash
set -e
cd "$(dirname "$0")"
go build -o engine ./cmd/engine
./engine "$@"
