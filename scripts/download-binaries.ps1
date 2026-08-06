# download-binaries.ps1
# Downloads all required binaries for TunnelCraft
# Run: powershell -ExecutionPolicy Bypass -File scripts/download-binaries.ps1

$ErrorActionPreference = "Stop"
$BinDir = Join-Path $PSScriptRoot "..\bin"
if (-not (Test-Path $BinDir)) { New-Item -ItemType Directory -Path $BinDir -Force | Out-Null }

$Downloads = @{
    # xray-core
    "xray-core.zip" = @{
        # Determine the latest version of xray-core from GitHub
        # Default to a known stable version
        Url   = "https://github.com/XTLS/Xray-core/releases/latest/download/Xray-windows-64.zip"
        File  = "xray-core.zip"
    }
    # wireguard-go
    "wireguard-go.exe" = @{
        Url   = "https://github.com/WireGuard/wireguard-go/releases/latest/download/wireguard-go-amd64.exe"
        File  = "wireguard-go.exe"
    }
    # amnezia-wg
    "amnezia-wg.exe" = @{
        Url   = "https://github.com/amnezia-vpn/amnezia-wg/releases/latest/download/amnezia-wg-windows-amd64.exe"
        File  = "amnezia-wg.exe"
    }
    # hysteria
    "hysteria-windows-amd64.zip" = @{
        Url   = "https://github.com/apernet/hysteria/releases/latest/download/hysteria-windows-amd64.zip"
        File  = "hysteria.zip"
    }
    # wintun
    "wintun.zip" = @{
        Url   = "https://www.wintun.net/builds/wintun-0.14.1.zip"
        File  = "wintun.zip"
    }
}

Write-Host "=== TunnelCraft Binary Downloader ===" -ForegroundColor Cyan
Write-Host ""

foreach ($key in $Downloads.Keys) {
    $item = $Downloads[$key]
    $outPath = Join-Path $BinDir $item.File
    $tempPath = Join-Path $env:TEMP $item.File

    Write-Host "[DOWNLOAD] $($item.File)" -ForegroundColor Yellow
    Write-Host "  URL: $($item.Url)"

    try {
        Invoke-WebRequest -Uri $item.Url -OutFile $tempPath -UseBasicParsing
        Write-Host "  Status: OK" -ForegroundColor Green

        # Handle zip archives
        if ($item.File -match "\.zip$") {
            $extractDir = Join-Path $env:TEMP $key.Replace(".zip", "")
            if (Test-Path $extractDir) { Remove-Item -Recurse -Force $extractDir }
            Expand-Archive -Path $tempPath -DestinationPath $extractDir -Force

            # Find and copy the main executable
            if ($key -eq "xray-core.zip") {
                $exe = Get-ChildItem -Path $extractDir -Filter "xray.exe" -Recurse | Select-Object -First 1
                if ($exe) {
                    Copy-Item $exe.FullName (Join-Path $BinDir "xray-core.exe") -Force
                    Write-Host "  Extracted: xray-core.exe" -ForegroundColor Green
                }
            } elseif ($key -eq "hysteria-windows-amd64.zip") {
                $exe = Get-ChildItem -Path $extractDir -Filter "hysteria*.exe" -Recurse | Select-Object -First 1
                if ($exe) {
                    Copy-Item $exe.FullName (Join-Path $BinDir "hysteria.exe") -Force
                    Write-Host "  Extracted: hysteria.exe" -ForegroundColor Green
                }
            } elseif ($key -eq "wintun.zip") {
                $dll = Get-ChildItem -Path $extractDir -Filter "wintun.dll" -Recurse | Select-Object -First 1
                if ($dll) {
                    Copy-Item $dll.FullName (Join-Path $BinDir "wintun.dll") -Force
                    Write-Host "  Extracted: wintun.dll" -ForegroundColor Green
                }
            }
            Remove-Item -Recurse -Force $extractDir
        } else {
            Copy-Item $tempPath $outPath -Force
            Write-Host "  Saved: $($item.File)" -ForegroundColor Green
        }
        Remove-Item -Force $tempPath
    } catch {
        Write-Host "  ERROR: Failed to download $($item.File): $_" -ForegroundColor Red
    }
    Write-Host ""
}

# Verify
Write-Host "=== Verification ===" -ForegroundColor Cyan
$required = @("xray-core.exe", "wireguard-go.exe", "amnezia-wg.exe", "hysteria.exe", "wintun.dll")
foreach ($f in $required) {
    $path = Join-Path $BinDir $f
    if (Test-Path $path) {
        $size = (Get-Item $path).Length
        Write-Host "  [OK] $f ($([math]::Round($size / 1MB, 2)) MB)" -ForegroundColor Green
    } else {
        Write-Host "  [MISSING] $f" -ForegroundColor Red
    }
}

Write-Host ""
Write-Host "Done." -ForegroundColor Cyan