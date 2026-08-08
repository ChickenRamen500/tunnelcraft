package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	protos "github.com/ChickenRamen500/tunnelcraft/core/internal/proto"
)

const (
	daemonAddr = "127.0.0.1:50051"
	guiAddr    = "127.0.0.1:8080"
)

var (
	tunnelClient protos.TunnelServiceClient
	serverClient protos.ServerServiceClient
)

func main() {
	conn, err := grpc.Dial(daemonAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("cannot connect to daemon: %v", err)
	}
	defer conn.Close()
	tunnelClient = protos.NewTunnelServiceClient(conn)
	serverClient = protos.NewServerServiceClient(conn)

	http.HandleFunc("/api/servers", handleServers)
	http.HandleFunc("/api/import", handleImport)
	http.HandleFunc("/api/connect/", handleConnect)
	http.HandleFunc("/api/disconnect", handleDisconnect)
	http.HandleFunc("/api/status", handleStatus)
	http.HandleFunc("/api/delete/", handleDelete)
	http.HandleFunc("/", handleIndex)

	go openBrowser(fmt.Sprintf("http://%s/", guiAddr))

	log.Printf("tcgui listening on %s", guiAddr)
	log.Fatal(http.ListenAndServe(guiAddr, nil))
}

func openBrowser(url string) {
	time.Sleep(500 * time.Millisecond)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Start()
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, indexHTML)
}

type ServerInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     uint32 `json:"port"`
}

type ImportRequest struct {
	File    string `json:"file,omitempty"`
	Content string `json:"content,omitempty"`
}

type ImportResponse struct {
	ID string `json:"id"`
	OK bool   `json:"ok"`
}

type ConnectResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type StatusResponse struct {
	State    string `json:"state"`
	ServerID string `json:"serverId"`
	Error    string `json:"error"`
}

type OKResponse struct {
	OK bool `json:"ok"`
}

func protocolName(p protos.Protocol) string {
	switch p {
	case protos.Protocol_PROTOCOL_VLESS:
		return "vless"
	case protos.Protocol_PROTOCOL_VMESS:
		return "vmess"
	case protos.Protocol_PROTOCOL_TROJAN:
		return "trojan"
	case protos.Protocol_PROTOCOL_SHADOWSOCKS:
		return "shadowsocks"
	case protos.Protocol_PROTOCOL_WIREGUARD:
		return "wireguard"
	case protos.Protocol_PROTOCOL_HYSTERIA:
		return "hysteria"
	case protos.Protocol_PROTOCOL_AMNEZIAWG:
		return "amneziawg"
	case protos.Protocol_PROTOCOL_SSH:
		return "ssh"
	default:
		return "unknown"
	}
}

func handleServers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := serverClient.ListServers(ctx, &protos.ListServersRequest{})
	if err != nil {
		// Return empty list instead of 500
		json.NewEncoder(w).Encode([]ServerInfo{})
		return
	}
	servers := make([]ServerInfo, len(resp.Servers))
	for i, s := range resp.Servers {
		name := s.Name
		if name == "" {
			name = "(no name)"
		}
		servers[i] = ServerInfo{ID: s.Id, Name: name, Protocol: protocolName(s.Protocol), Host: s.Host, Port: s.Port}
	}
	json.NewEncoder(w).Encode(servers)
}

func handleImport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req ImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	var content string
	if req.File != "" {
		data, err := os.ReadFile(req.File)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		content = string(data)
	} else if req.Content != "" {
		content = req.Content
	} else {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "file or content required"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := serverClient.ImportServers(ctx, &protos.ImportServersRequest{Content: content})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	var id string
	if len(resp.ImportedServerIds) > 0 {
		id = resp.ImportedServerIds[0]
	}
	json.NewEncoder(w).Encode(ImportResponse{ID: id, OK: true})
}

func handleConnect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/connect/")
	if path == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "server id required"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	_, err := tunnelClient.Connect(ctx, &protos.ConnectRequest{ServerId: path})
	if err != nil {
		json.NewEncoder(w).Encode(ConnectResponse{OK: false, Error: err.Error()})
		return
	}
	json.NewEncoder(w).Encode(ConnectResponse{OK: true})
}

func handleDisconnect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := tunnelClient.Disconnect(ctx, &protos.DisconnectRequest{})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(OKResponse{OK: true})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := tunnelClient.GetConnectionStatus(ctx, &protos.GetConnectionStatusRequest{})
	if err != nil {
		// Don't return 500 — daemon might be temporarily unreachable
		log.Printf("[tcgui] GetConnectionStatus error: %v", err)
		json.NewEncoder(w).Encode(StatusResponse{State: "disconnected", Error: ""})
		return
	}
	log.Printf("[tcgui] status: state=%d, server=%s", resp.State, resp.ServerId)
	stateStr := "disconnected"
	switch resp.State {
	case protos.ConnectionState_CONNECTION_STATE_DISCONNECTED:
		stateStr = "disconnected"
	case protos.ConnectionState_CONNECTION_STATE_CONNECTING:
		stateStr = "connecting"
	case protos.ConnectionState_CONNECTION_STATE_CONNECTED:
		stateStr = "connected"
	case protos.ConnectionState_CONNECTION_STATE_DISCONNECTING:
		stateStr = "disconnecting"
	case protos.ConnectionState_CONNECTION_STATE_ERROR:
		stateStr = "error"
	}
	json.NewEncoder(w).Encode(StatusResponse{State: stateStr, ServerID: resp.ServerId, Error: ""})
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "DELETE" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/delete/")
	if path == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "server id required"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := serverClient.DeleteServer(ctx, &protos.DeleteServerRequest{Id: path})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(OKResponse{OK: true})
}

const indexHTML = `<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="UTF-8">
<title>TunnelCraft Control</title>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: 'Segoe UI', monospace; background: #1a1a2e; color: #e0e0e0; padding: 20px; }
.container { max-width: 820px; margin: 0 auto; border: 1px solid #333; border-radius: 8px; overflow: hidden; box-shadow: 0 4px 24px rgba(0,0,0,0.4); }
.header { background: #16213e; padding: 16px 24px; display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid #0f3460; }
.header h1 { font-size: 18px; color: #e94560; letter-spacing: 1px; }
.conn-info { display: flex; align-items: center; gap: 10px; font-size: 14px; }
.dot { width: 10px; height: 10px; border-radius: 50%; background: #555; transition: background 0.3s; }
.dot.connected { background: #4caf50; box-shadow: 0 0 8px #4caf5066; }
.dot.connecting { background: #ff9800; animation: pulse 1s infinite; }
.dot.error { background: #f44336; }
@keyframes pulse { 0%,100% { opacity: 1; } 50% { opacity: 0.3; } }
.content { padding: 20px; }
.section { margin-bottom: 18px; }
.section-label { font-size: 12px; color: #888; text-transform: uppercase; letter-spacing: 1px; margin-bottom: 8px; }
.import-row { display: flex; gap: 10px; }
.import-input { flex: 1; background: #0f3460; border: 1px solid #1a1a5e; color: #e0e0e0; padding: 10px 14px; font-family: inherit; font-size: 14px; border-radius: 6px; outline: none; transition: border-color 0.2s; }
.import-input:focus { border-color: #e94560; }
button { background: #16213e; border: 1px solid #0f3460; color: #e0e0e0; padding: 10px 18px; cursor: pointer; border-radius: 6px; font-size: 13px; font-family: inherit; transition: all 0.2s; }
button:hover { background: #1a1a5e; border-color: #e94560; }
button:active { transform: scale(0.97); }
button.primary { background: #e94560; border-color: #e94560; color: #fff; font-weight: 600; }
button.primary:hover { background: #c73652; }
button.danger { background: #5c1a1a; border-color: #7a2a2a; }
button.danger:hover { background: #7a2a2a; }
.servers-list { background: #0f1a2e; border: 1px solid #1a1a5e; border-radius: 6px; max-height: 320px; overflow-y: auto; }
.server-item { display: flex; align-items: center; padding: 12px 16px; border-bottom: 1px solid #1a1a3e; cursor: pointer; transition: background 0.15s; }
.server-item:last-child { border-bottom: none; }
.server-item:hover { background: #16213e; }
.server-item.selected { background: #1a2744; border-left: 3px solid #e94560; }
.server-radio { width: 16px; height: 16px; margin-right: 14px; accent-color: #e94560; }
.server-info { flex: 1; min-width: 0; }
.server-name { font-size: 14px; color: #fff; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.server-meta { font-size: 12px; color: #666; margin-top: 3px; }
.server-connect-btn { background: #0f3460; border: 1px solid #1a4a7a; color: #4fc3f7; padding: 6px 14px; font-size: 12px; border-radius: 4px; cursor: pointer; font-family: inherit; transition: all 0.2s; }
.server-connect-btn:hover { background: #1a4a7a; color: #fff; }
.actions-row { display: flex; gap: 10px; margin-top: 14px; flex-wrap: wrap; }
.log-container { background: #0a0a1a; border: 1px solid #1a1a3e; border-radius: 6px; padding: 12px; height: 180px; overflow-y: auto; font-size: 12px; font-family: 'Cascadia Code', 'Consolas', monospace; }
.log-entry { margin-bottom: 3px; color: #666; line-height: 1.5; }
.log-entry .ts { color: #4fc3f7; }
.log-entry .ok { color: #4caf50; }
.log-entry .err { color: #f44336; }
.status-bar { background: #16213e; padding: 12px 24px; border-top: 1px solid #0f3460; font-size: 13px; display: flex; gap: 24px; }
.status-bar-item { color: #666; }
.status-bar-item strong { color: #e0e0e0; }
.empty-msg { padding: 30px; text-align: center; color: #444; font-size: 14px; }
</style>
</head>
<body>
<div class="container">
  <div class="header">
    <h1>TunnelCraft</h1>
    <div class="conn-info">
      <span id="headerStatus">--</span>
      <div class="dot" id="headerDot"></div>
    </div>
  </div>

  <div class="content">
    <div class="section">
      <div class="section-label">Импорт конфига</div>
      <div class="import-row">
        <input type="text" class="import-input" id="importPath" placeholder="C:\path\to\config.conf">
        <button class="primary" onclick="doImport()">Импорт</button>
      </div>
    </div>

    <div class="section">
      <div class="section-label">Серверы</div>
      <div class="servers-list" id="serversList">
        <div class="empty-msg">Загрузка...</div>
      </div>
    </div>

    <div class="actions-row">
      <button class="danger" onclick="doDisconnect()">Отключить</button>
      <button onclick="doRefresh()">Обновить</button>
      <button onclick="doDelete()">Удалить выбранный</button>
    </div>

    <div class="section" style="margin-top:18px">
      <div class="section-label">Лог</div>
      <div class="log-container" id="logBox"></div>
    </div>
  </div>

  <div class="status-bar">
    <div class="status-bar-item">Статус: <strong id="stState">--</strong></div>
    <div class="status-bar-item">Сервер: <strong id="stServer">--</strong></div>
  </div>
</div>

<script>
var selectedId = null;

function log(msg, type) {
  var box = document.getElementById('logBox');
  var now = new Date().toLocaleTimeString('ru-RU');
  var div = document.createElement('div');
  div.className = 'log-entry';
  var cls = type === 'ok' ? 'ok' : type === 'err' ? 'err' : '';
  div.innerHTML = '<span class="ts">[' + now + ']</span> <span class="' + cls + '">' + msg + '</span>';
  box.appendChild(div);
  box.scrollTop = box.scrollHeight;
}

function api(url, opts) {
  opts = opts || {};
  opts.headers = Object.assign({'Content-Type': 'application/json'}, opts.headers || {});
  return fetch(url, opts).then(function(r) { return r.json(); });
}

function renderServers(servers) {
  var el = document.getElementById('serversList');
  if (!servers || servers.length === 0) {
    el.innerHTML = '<div class="empty-msg">Нет серверов</div>';
    return;
  }
  var html = '';
  for (var i = 0; i < servers.length; i++) {
    var s = servers[i];
    var sel = s.id === selectedId ? ' selected' : '';
    var chk = s.id === selectedId ? ' checked' : '';
    html += '<div class="server-item' + sel + '" onclick="selectSv(\'' + s.id + '\')">'
      + '<input type="radio" class="server-radio" name="sv"' + chk + ' onclick="event.stopPropagation();selectSv(\'' + s.id + '\')">'
      + '<div class="server-info">'
      + '<div class="server-name">' + (s.name || '(no name)') + '</div>'
      + '<div class="server-meta">' + s.protocol + ' &mdash; ' + s.host + ':' + s.port + '</div>'
      + '</div>'
      + '<button class="server-connect-btn" onclick="event.stopPropagation();doConnect(\'' + s.id + '\')">Connect</button>'
      + '</div>';
  }
  el.innerHTML = html;
}

function selectSv(id) {
  selectedId = id;
  loadServers();
}

function loadServers() {
  api('/api/servers').then(function(data) {
    renderServers(data);
  }).catch(function(e) {
    document.getElementById('serversList').innerHTML = '<div class="empty-msg" style="color:#f44336">Ошибка загрузки серверов</div>';
    log('Ошибка: ' + e.message, 'err');
  });
}

function doImport() {
  var path = document.getElementById('importPath').value.trim();
  if (!path) { log('Введите путь к файлу', 'err'); return; }
  log('Импорт: ' + path);
  api('/api/import', {
    method: 'POST',
    body: JSON.stringify({file: path})
  }).then(function(r) {
    if (r.ok) {
      log('Импортировано: ' + (r.id || 'OK'), 'ok');
      document.getElementById('importPath').value = '';
      loadServers();
    } else {
      log('Ошибка импорта: ' + (r.error || '?'), 'err');
    }
  }).catch(function(e) { log('Ошибка: ' + e.message, 'err'); });
}

function doConnect(id) {
  log('Подключение...');
  api('/api/connect/' + id, {method: 'POST'}).then(function(r) {
    if (r.ok) {
      log('Подключено', 'ok');
    } else {
      log('Ошибка: ' + (r.error || '?'), 'err');
    }
    updateStatus();
  }).catch(function(e) { log('Ошибка: ' + e.message, 'err'); });
}

function doDisconnect() {
  log('Отключение...');
  api('/api/disconnect', {method: 'POST'}).then(function() {
    log('Отключено', 'ok');
    updateStatus();
  }).catch(function(e) { log('Ошибка: ' + e.message, 'err'); });
}

function doDelete() {
  if (!selectedId) { log('Выберите сервер', 'err'); return; }
  log('Удаление...');
  api('/api/delete/' + selectedId, {method: 'DELETE'}).then(function() {
    log('Удалено', 'ok');
    selectedId = null;
    loadServers();
  }).catch(function(e) { log('Ошибка: ' + e.message, 'err'); });
}

function doRefresh() {
  log('Обновление...');
  loadServers();
  updateStatus();
}

function updateStatus() {
  api('/api/status').then(function(st) {
    console.log('[tcgui] status response:', JSON.stringify(st));
    var state = st.state || 'disconnected';
    document.getElementById('stState').textContent = state;
    document.getElementById('stServer').textContent = st.serverId || '--';
    var hs = document.getElementById('headerStatus');
    var hd = document.getElementById('headerDot');
    hd.className = 'dot ' + state;
    if (state === 'connected') hs.textContent = 'Подключено';
    else if (state === 'connecting') hs.textContent = 'Подключение...';
    else if (state === 'error') hs.textContent = 'Ошибка';
    else hs.textContent = 'Отключено';
  }).catch(function() {
    document.getElementById('stState').textContent = 'error';
    document.getElementById('headerDot').className = 'dot error';
    document.getElementById('headerStatus').textContent = 'Нет связи';
  });
}

log('TunnelCraft Control запущен', 'ok');
loadServers();
updateStatus();
setInterval(updateStatus, 5000);
</script>
</body>
</html>`
