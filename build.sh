#!/bin/bash
set -e

echo "Building Auth Service..."
cd backend
go mod download
go build -tags netgo -ldflags '-s -w' -o ../app ./services/auth/cmd/server/...
echo "Build complete!"