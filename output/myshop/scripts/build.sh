#!/bin/bash
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/../srvProduct" && go build -o ../../bin/srvProduct ./cmd/main.go
echo "Build done"
