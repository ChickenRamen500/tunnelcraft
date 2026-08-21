# download-binaries.ps1
# Downloads all required binaries for TunnelCraft
# Run: powershell -ExecutionPolicy Bypass -File scripts\download-binaries.ps1

$ErrorActionPreference = "Continue"  # don't abort on first error
$BinDir = Join-Path $PSScriptRoot "..\bin"
if (-not (Test-Path $BinDir)) { New-Item -ItemType Directory -Path $BinDir -Force | Out-Null }

Write-Host "=== TunnelCraft Binary Downloader ===" -ForegroundColor Cyan
Write-Host ""

# ------------------------------------------------------------------
# 1. xray-core.exe  — GitHub releases (WORKS)
# ------------------------------------------------------------------
Write-Host "[1/5] xray-core.exe" -ForegroundColor Yellow
try {
    $tmp = Join-Path $env:TEMP "tunnelcraft-xray.zip"
    Invoke-WebRequest -Uri "https://github.com/XTLS/Xray-core/releases/latest/download/Xray-windows-64.zip" -OutFile $tmp -UseBasicParsing
    $ext = Join-Path $env:TEMP "tunnelcraft-xray"
    if (Test-Path $ext) { Remove-Item -Recurse -Force $ext }
    Expand-Archive -Path $tmp -DestinationPath $ext -Force
    $exe = Get-ChildItem -Path $ext -Filter "xray.exe" -Recurse | Select-Object -First 1
    if ($exe) {
        Copy-Item $exe.FullName (Join-Path $BinDir "xray-core.exe") -Force
        $sz = [math]::Round((Get-Item (Join-Path $BinDir "xray-core.exe")).Length / 1MB, 2)
        Write-Host "  [OK] xray-core.exe ($sz MB)" -ForegroundColor Green
    } else {
        Write-Host "  [WARN] xray.exe not found inside zip" -ForegroundColor Red
    }
    Remove-Item -Recurse -Force $ext
    Remove-Item -Force $tmp
} catch {
    Write-Host "  [FAIL] $($_.Exception.Message)" -ForegroundColor Red
}
Write-Host ""

# ------------------------------------------------------------------
# 2. hysteria.exe  — GitHub releases (standalone exe, not zip)
# ------------------------------------------------------------------
Write-Host "[2/5] hysteria.exe" -ForegroundColor Yellow
try {
    Invoke-WebRequest -Uri "https://github.com/apernet/hysteria/releases/latest/download/hysteria-windows-amd64.exe" -OutFile (Join-Path $BinDir "hysteria.exe") -UseBasicParsing
    $sz = [math]::Round((Get-Item (Join-Path $BinDir "hysteria.exe")).Length / 1MB, 2)
    Write-Host "  [OK] hysteria.exe ($sz MB)" -ForegroundColor Green
} catch {
    Write-Host "  [FAIL] $($_.Exception.Message)" -ForegroundColor Red
}
Write-Host ""

# ------------------------------------------------------------------
# 3. wintun.dll  — wintun.net (WORKS)
# ------------------------------------------------------------------
Write-Host "[3/5] wintun.dll" -ForegroundColor Yellow
try {
    $tmp = Join-Path $env:TEMP "tunnelcraft-wintun.zip"
    Invoke-WebRequest -Uri "https://www.wintun.net/builds/wintun-0.14.1.zip" -OutFile $tmp -UseBasicParsing
    $ext = Join-Path $env:TEMP "tunnelcraft-wintun"
    if (Test-Path $ext) { Remove-Item -Recurse -Force $ext }
    Expand-Archive -Path $tmp -DestinationPath $ext -Force
    $dll = Get-ChildItem -Path $ext -Filter "wintun.dll" -Recurse | Select-Object -First 1
    if ($dll) {
        Copy-Item $dll.FullName (Join-Path $BinDir "wintun.dll") -Force
        $sz = [math]::Round((Get-Item (Join-Path $BinDir "wintun.dll")).Length / 1KB, 2)
        Write-Host "  [OK] wintun.dll ($sz KB)" -ForegroundColor Green
    } else {
        Write-Host "  [WARN] wintun.dll not found inside zip" -ForegroundColor Red
    }
    Remove-Item -Recurse -Force $ext
    Remove-Item -Force $tmp
} catch {
    Write-Host "  [FAIL] $($_.Exception.Message)" -ForegroundColor Red
}
Write-Host ""

# ------------------------------------------------------------------
# 2.5. wireguard.exe  — copy from system WireGuard installation
# ------------------------------------------------------------------
Write-Host "[2.5/5] wireguard.exe" -ForegroundColor Yellow
$wgExeSystem = @(
    "C:\Program Files\WireGuard\wireguard.exe",
    "C:\Program Files (x86)\WireGuard\wireguard.exe"
)
$wgExeDst = Join-Path $BinDir "wireguard.exe"
$wgCopied = $false
foreach ($p in $wgExeSystem) {
    if (Test-Path $p) {
        Copy-Item $p $wgExeDst -Force
        $sz = [math]::Round((Get-Item $wgExeDst).Length / 1MB, 2)
        Write-Host "  [OK] wireguard.exe copied from $p ($sz MB)" -ForegroundColor Green
        $wgCopied = $true
        break
    }
}
if (-not $wgCopied) {
    Write-Host "  [SKIP] wireguard.exe not found on system." -ForegroundColor DarkYellow
    Write-Host "         Install WireGuard from https://www.wireguard.com/install/" -ForegroundColor DarkYellow
    Write-Host "         Then re-run this script." -ForegroundColor DarkYellow
}
Write-Host ""

# ------------------------------------------------------------------
# 4. wireguard-go.exe  — extracted from WireGuard for Windows installer
#    The WireGuard repo has NO releases. We download the official MSI
#    and extract wireguard-go.exe from it.
# ------------------------------------------------------------------
Write-Host "[4/5] wireguard-go.exe" -ForegroundColor Yellow
$wgExe = Join-Path $BinDir "wireguard-go.exe"

# Check if already installed on system
 $wgSystemPaths = @(
    "C:\Program Files\WireGuard\wireguard-go.exe",
    "C:\Program Files\WireGuard\wg.exe",
    "C:\Program Files (x86)\WireGuard\wireguard-go.exe",
    "C:\Program Files (x86)\WireGuard\wg.exe"
)
$found = $false
foreach ($p in $wgSystemPaths) {
    if (Test-Path $p) {
        Copy-Item $p $wgExe -Force
        $sz = [math]::Round((Get-Item $wgExe).Length / 1MB, 2)
        Write-Host "  [OK] wireguard-go.exe copied from $p ($sz MB)" -ForegroundColor Green
        $found = $true
        break
    }
}

if (-not $found) {
    # Try downloading MSI and extracting
    Write-Host "  WireGuard not found on system. Downloading MSI..." -ForegroundColor DarkGray
    try {
        $msiTmp = Join-Path $env:TEMP "tunnelcraft-wireguard.msi"
        Invoke-WebRequest -Uri "https://download.wireguard.com/windows-client/wireguard-installer.exe" -OutFile $msiTmp -UseBasicParsing

        # The installer is an .exe (Inno Setup), not MSI.
        # Try to run it with /SILENT /DIR to extract, or use 7zip if available
        $sevenZip = Get-Command 7z -ErrorAction SilentlyContinue
        if ($sevenZip) {
            $ext = Join-Path $env:TEMP "tunnelcraft-wg-extract"
            if (Test-Path $ext) { Remove-Item -Recurse -Force $ext }
            & 7z x $msiTmp -o$ext -y | Out-Null
            $foundExe = Get-ChildItem -Path $ext -Filter "wireguard-go.exe" -Recurse | Select-Object -First 1
            if ($foundExe) {
                Copy-Item $foundExe.FullName $wgExe -Force
                $sz = [math]::Round((Get-Item $wgExe).Length / 1MB, 2)
                Write-Host "  [OK] wireguard-go.exe extracted ($sz MB)" -ForegroundColor Green
                $found = $true
            }
            Remove-Item -Recurse -Force $ext
        } else {
            Write-Host "  Could not extract. Install 7-Zip or WireGuard manually." -ForegroundColor DarkYellow
        }
        Remove-Item -Force $msiTmp
    } catch {
        Write-Host "  [FAIL] $($_.Exception.Message)" -ForegroundColor Red
    }
}

# Try building from source if Go is available
if (-not (Test-Path $wgExe)) {
    $goCmd = Get-Command go -ErrorAction SilentlyContinue
    if ($goCmd) {
        Write-Host "  Attempting to build wireguard-go from source..." -ForegroundColor DarkGray
        try {
            $env:GOOS = "windows"
            $env:GOARCH = "amd64"
            & go install golang.zx2c4.com/wireguard/cmd/wireguard-go@latest 2>&1 | Out-Null
            $goPath = (& go env GOPATH 2>$null)
            if ($goPath) {
                $builtExe = Join-Path $goPath "bin\wireguard-go.exe"
                if (Test-Path $builtExe) {
                    Copy-Item $builtExe $wgExe -Force
                    $sz = [math]::Round((Get-Item $wgExe).Length / 1MB, 2)
                    Write-Host "  [OK] wireguard-go.exe built from source ($sz MB)" -ForegroundColor Green
                    $found = $true
                }
            }
        } catch {
            Write-Host "  [WARN] go build failed: $($_.Exception.Message)" -ForegroundColor DarkYellow
        }
    }
}

if (-not (Test-Path $wgExe)) {
    Write-Host "  [SKIP] wireguard-go.exe not available. WG/AWG won't work yet." -ForegroundColor DarkYellow
    Write-Host "         Fix: Install WireGuard from https://www.wireguard.com/install/" -ForegroundColor DarkYellow
    Write-Host "         or:  go install golang.zx2c4.com/wireguard/cmd/wireguard-go@latest" -ForegroundColor DarkYellow
    Write-Host "         then re-run this script." -ForegroundColor DarkYellow
}
Write-Host ""

# ------------------------------------------------------------------
# 5. amnezia-wg.exe — AmneziaWG binary (AWG2 + AWG3 support)
#    Clone and build from amnezia-vpn/amneziawg-windows repository
# ------------------------------------------------------------------
Write-Host "[5/5] amnezia-wg.exe" -ForegroundColor Yellow
$awgExe = Join-Path $BinDir "amnezia-wg.exe"
$awgFound = $false

# Check if already in bin/
if (Test-Path $awgExe) {
    $sz = [math]::Round((Get-Item $awgExe).Length / 1MB, 2)
    Write-Host "  [OK] amnezia-wg.exe already in bin/ ($sz MB)" -ForegroundColor Green
    $awgFound = $true
}

# Try copying from AmneziaVPN installation
if (-not $awgFound) {
    $awgSystemPaths = @(
        "C:\Program Files\AmneziaVPN\amnezia-wg.exe",
        "C:\Program Files\AmneziaVPN\resources\amnezia-wg.exe",
        "C:\Program Files (x86)\AmneziaVPN\amnezia-wg.exe"
    )
    foreach ($p in $awgSystemPaths) {
        if (Test-Path $p) {
            Copy-Item $p $awgExe -Force
            $sz = [math]::Round((Get-Item $awgExe).Length / 1MB, 2)
            Write-Host "  [OK] amnezia-wg.exe copied from AmneziaVPN install ($sz MB)" -ForegroundColor Green
            $awgFound = $true
            break
        }
    }
}

# Build from source by cloning the repository
if (-not $awgFound) {
    $goCmd = Get-Command go -ErrorAction SilentlyContinue
    if ($goCmd) {
        Write-Host "  Cloning and building amnezia-wg from source..." -ForegroundColor DarkGray
        try {
            $tempDir = Join-Path $env:TEMP "tunnelcraft-amnezia-build"
            if (Test-Path $tempDir) { Remove-Item -Recurse -Force $tempDir }
            New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
            
            & git clone --depth 1 https://github.com/amnezia-vpn/amneziawg-windows $tempDir 2>&1 | Out-Null
            
            if (Test-Path (Join-Path $tempDir "cmd\amnezia-wg")) {
                cd $tempDir\cmd\amnezia-wg
                $env:GOOS = "windows"
                $env:GOARCH = "amd64"
                & go build -o $awgExe . 2>&1 | Out-Null
                
                if (Test-Path $awgExe) {
                    $sz = [math]::Round((Get-Item $awgExe).Length / 1MB, 2)
                    Write-Host "  [OK] amnezia-wg.exe built from source ($sz MB)" -ForegroundColor Green
                    $awgFound = $true
                } else {
                    Write-Host "  [WARN] amnezia-wg build produced no binary" -ForegroundColor DarkYellow
                }
                cd $PSScriptRoot
            } else {
                Write-Host "  [WARN] amnezia-wg source structure unexpected" -ForegroundColor DarkYellow
            }
            
            Remove-Item -Recurse -Force $tempDir -ErrorAction SilentlyContinue
        } catch {
            Write-Host "  [WARN] amnezia-wg clone/build failed: $($_.Exception.Message)" -ForegroundColor DarkYellow
        }
    } else {
        Write-Host "  [SKIP] Go not installed, cannot build amnezia-wg" -ForegroundColor DarkYellow
    }
}

if (-not $awgFound) {
    Write-Host "  [SKIP] amnezia-wg.exe not found." -ForegroundColor DarkYellow
    Write-Host "         Options:" -ForegroundColor DarkYellow
    Write-Host "         1. Copy from AmneziaVPN install to bin\amnezia-wg.exe" -ForegroundColor DarkYellow
    Write-Host "         2. Install Go and re-run this script (will auto-build)" -ForegroundColor DarkYellow
    Write-Host "         3. Manually place amnezia-wg.exe in the bin/ folder" -ForegroundColor DarkYellow
}
Write-Host ""

# ------------------------------------------------------------------
# Summary
# ------------------------------------------------------------------
Write-Host "=== Summary ===" -ForegroundColor Cyan
$required = @("xray-core.exe", "hysteria.exe", "wintun.dll", "wireguard.exe", "amnezia-wg.exe")
$ok = 0
$fail = 0
foreach ($f in $required) {
    $path = Join-Path $BinDir $f
    if (Test-Path $path) {
        $size = (Get-Item $path).Length
        Write-Host "  [OK]     $f ($([math]::Round($size / 1MB, 2)) MB)" -ForegroundColor Green
        $ok++
    } else {
        Write-Host "  [MISSING] $f" -ForegroundColor Red
        $fail++
    }
}
Write-Host ""
if ($fail -eq 0) {
    Write-Host "All binaries ready! You can start tunnelcraftd now." -ForegroundColor Green
} else {
    Write-Host "$ok ok, $fail missing. Core features (xray, hysteria) should work." -ForegroundColor Yellow
    if ($fail -le 2) {
        Write-Host "Note: WireGuard/AmneziaWG features may not work until binaries are installed." -ForegroundColor DarkYellow
    }
}
