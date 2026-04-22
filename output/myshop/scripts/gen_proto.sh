#!/bin/bash
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
mkdir -p "$PROJECT_DIR/common/kitexGen"
cd "$PROJECT_DIR/common/idl"
protoc --go_out=../../common/kitexGen --go_opt=module=myshop/common/kitexGen --go-grpc_out=../../common/kitexGen --go-grpc_opt=module=myshop/common/kitexGen product.proto
echo "Done"
