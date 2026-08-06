#!/usr/bin/env bash
# generate-proto.sh — Generate Go code from .proto files
# Prerequisites: protoc, protoc-gen-go, protoc-gen-go-grpc
# Run: bash scripts/generate-proto.sh

set -euo pipefail

PROTO_DIR="proto"
OUT_DIR="core/internal/proto"

# Ensure output directory exists
mkdir -p "$OUT_DIR"

# Generate Go code
protoc \
  --proto_path="$PROTO_DIR" \
  --go_out="$OUT_DIR" --go_opt=paths=source_relative \
  --go-grpc_out="$OUT_DIR" --go-grpc_opt=paths=source_relative \
  "$PROTO_DIR/tunnelcraft.proto"

echo "Generated proto Go code in $OUT_DIR/"
