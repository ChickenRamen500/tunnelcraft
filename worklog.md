# TunnelCraft Worklog

---
Task ID: 1
Agent: main
Task: Fix build error in parser.go (non-boolean condition + missing return values)

Work Log:
- Identified that `parseProviderJSONLinks` returns `([]engine.ServerConfig, []ParseError)` (no bool)
- Line 165 used `if links, ok := parseProviderJSONLinks(payload); ok` — `ok` was `[]ParseError`, not `bool`
- Line 166 `return links` only returned 1 value, needed 2
- Fixed by extracting both values and checking `links != nil`

Stage Summary:
- Fixed parser.go:165-168 to properly call parseProviderJSONLinks and return both values

---
Task ID: 2
Agent: main
Task: Fix REALITY settings not parsed from Xray JSON configs

Work Log:
- Discovered that in real xray-core configs, `realitySettings` is at `streamSettings` level, NOT inside `tlsSettings`
- The parser struct `xrayStreamSettings` only had `RealitySettings` inside `xrayTLSSettings`
- Added `RealitySettings *xrayRealitySettings` field to `xrayStreamSettings` struct
- Updated parsing code to check `ss.RealitySettings` at streamSettings level first
- Kept backward compat with `ss.TLSSettings.RealitySettings` (legacy placement)

Stage Summary:
- Fixed xrayStreamSettings struct to include realitySettings at correct level
- Fixed parsing logic to check both placements

---
Task ID: 3
Agent: main
Task: Fix xray HTTP inbound with port 0 in bridge mode

Work Log:
- Bridge mode calls `xray.Start(ctx, server, b.bridgePort, 0)` — httpPort=0
- Xray config generated an HTTP inbound with port 0, which is invalid
- Extracted inbound building into `buildInbounds()` method
- Method skips HTTP inbound when httpPort is 0

Stage Summary:
- Added `buildInbounds()` method to XrayHandler
- HTTP inbound omitted when port is 0 (bridge mode)

---
Task ID: 4
Agent: main
Task: Fix DNS not working through TUN (ERR_NAME_NOT_RESOLVED)

Work Log:
- Identified circular dependency: proxy needs DNS → dns-remote → DoH → proxy (loop!)
- `dns-remote` had no `detour`, so DoH connection used default route (through proxy)
- Proxy outbound's server domain DNS also went through dns-remote (circular)
- Fix 1: Added `detour: proxyTag` to `dns-remote` so DoH explicitly uses the proxy
- Fix 2: Added DNS rule for server's domain → `dns-local` (direct, system DNS)
- This breaks the circular dependency: server domain resolved directly, DoH goes through tunnel
- Updated `buildDNS` signature to accept `proxyTag` and `server` parameters
- Updated both callers: non-bridge (`proxy-main`) and bridge (`proxy-bridge`)

Stage Summary:
- Fixed DNS circular dependency by adding explicit detour and server domain rule
- `buildDNS` now takes `proxyTag` and `server` parameters

---
Task ID: 5
Agent: main
Task: Add PING feature to server list (per-server ping + ping all)

Work Log:
- Backend: Added `GET /api/ping?id=xxx` endpoint to `http_server.go`
  - Looks up server by ID from config, calls existing `engine.TCPPing()` (1 attempt, 2s timeout)
  - Returns `{ id, latency_ms, ok, error? }` JSON response
  - Registered route in the mux alongside other API routes
- Frontend API: Added `PingResult` interface and `pingServer(id)` function to `useApi.ts`
- Frontend ServerCard: Added per-server ping button (Activity icon) with three states:
  - Idle: gray Activity icon button
  - Pinging: pulsing yellow Activity icon
  - Result: color-coded latency text (green <100ms, yellow <300ms, orange <800ms, red >=800ms/timeout)
  - Button has `stopPropagation` so it doesn't trigger server selection
- Frontend Servers page: Added "Пинг всех" (Ping All) button in header
  - Fires all ping requests concurrently via `Promise.allSettled`
  - Tracks pinging state per-server via `Set<string>` of IDs
  - Button disabled while pinging or when no servers exist
  - Fixed favorites filter bug: previously protocol filter ran before favorites override, now they're mutually exclusive

Stage Summary:
- Backend: new `/api/ping` endpoint using existing `engine.TCPPing`
- Frontend: per-server ping button with color-coded latency display
- Frontend: "Ping All" button fires concurrent pings to all servers
- Ping state managed locally in Servers component (not global store)
- Minor fix: favorites filter now correctly mutually exclusive with protocol filter