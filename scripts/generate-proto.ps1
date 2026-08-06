# generate-proto.ps1 — Generate Go code from .proto files (Windows)
# Prerequisites: protoc, protoc-gen-go, protoc-gen-go-grpc
# Run: powershell -ExecutionPolicy Bypass -File scripts/generate-proto.ps1

$ErrorActionPreference = "Stop"

$ProtoDir = Join-Path $PSScriptRoot "..\proto"
$OutDir = Join-Path $PSScriptRoot "..\core\internal\proto"

if (-not (Test-Path $OutDir)) {
    New-Item -ItemType Directory -Path $OutDir -Force | Out-Null
}

& protoc `
    --proto_path="$ProtoDir" `
    --go_out="$OutDir" --go_opt=paths=source_relative `
    --go-grpc_out="$OutDir" --go-grpc_opt=paths=source_relative `
    (Join-Path $ProtoDir "tunnelcraft.proto")

Write-Host "Generated proto Go code in $OutDir/" -ForegroundColor Green
