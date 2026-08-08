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
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
resp, err := serverClient.ListServers(ctx, &protos.ListServersRequest{})
if err != nil {
w.WriteHeader(http.StatusInternalServerError)
json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
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
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
resp, err := tunnelClient.GetConnectionStatus(ctx, &protos.GetConnectionStatusRequest{})
if err != nil {
w.WriteHeader(http.StatusInternalServerError)
json.NewEncoder(w).Encode(StatusResponse{State: "error", Error: err.Error()})
return
}
stateStr := "disconnected"
switch resp.State {
case protos.ConnectionState_CONNECTION_STATE_DISCONNECTED:
stateStr = "disconnected"
case protos.ConnectionState_CONNECTION_STATE_CONNECTING:
stateStr = "connecting"
case protos.ConnectionState_CONNECTION_STATE_CONNECTED:
stateStr = "connected"
case protos.ConnectionState_CONNECTION_STATE_RECONNECTING:
stateStr = "reconnecting"
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

const indexHTML = "<!DOCTYPE html><html lang=\"ru\"><head><meta charset=\"UTF-8\"><title>TunnelCraft Control</title><style>*{margin:0;padding:0;box-sizing:border-box}body{font-family:monospace;background:#1a1a1a;color:#c0c0c0;padding:20px}.container{max-width:800px;margin:0 auto;border:1px solid #333;border-radius:4px;overflow:hidden}.header{background:#2a2a2a;padding:15px 20px;display:flex;justify-content:space-between;align-items:center;border-bottom:1px solid #333}.header h1{font-size:18px;color:#fff}.status-dot{width:10px;height:10px;border-radius:50%;background:#444}.status-dot.connected{background:#4caf50}.status-dot.connecting{background:#ff9800}.status-dot.disconnected{background:#f44336}.status-dot.error{background:#9c27b0}.content{padding:20px}.section{margin-bottom:20px}.section-label{font-size:14px;color:#888;margin-bottom:8px}.import-row{display:flex;gap:10px}.import-input{flex:1;background:#2a2a2a;border:1px solid #444;color:#c0c0c0;padding:8px 12px;font-family:inherit;border-radius:3px}button{background:#3a3a3a;border:1px solid #555;color:#c0c0c0;padding:8px 16px;cursor:pointer;border-radius:3px}button:hover{background:#4a4a4a}button.primary{background:#2d5a27;border-color:#3d6a37}button.danger{background:#5a2d27;border-color:#6a3d37}.servers-list{background:#2a2a2a;border:1px solid #333;border-radius:3px;max-height:300px;overflow-y:auto}.server-item{display:flex;align-items:center;padding:12px 15px;border-bottom:1px solid #333;cursor:pointer}.server-item:hover{background:#333}.server-item.selected{background:#3a3a4a}.server-radio{width:16px;height:16px;margin-right:12px}.server-info{flex:1}.server-name{font-size:14px;color:#fff}.server-meta{font-size:12px;color:#888;margin-top:2px}.server-actions{display:flex;gap:8px}.actions-row{display:flex;gap:10px;margin-top:15px}.log-container{background:#0a0a0a;border:1px solid #333;border-radius:3px;padding:10px;height:200px;overflow-y:auto;font-size:12px}.log-entry{margin-bottom:4px;color:#888}.log-entry span{color:#4caf50}.status-bar{background:#2a2a2a;padding:10px 20px;border-top:1px solid #333;font-size:13px;display:flex;gap:20px}.status-bar-item{color:#888}.status-bar-item strong{color:#c0c0c0}</style></head><body><div class=\"container\"><div class=\"header\"><h1>TunnelCraft Control</h1><div class=\"connection-status\"><span id=\"headerStatus\">Подключение...</span><div class=\"status-dot\" id=\"headerDot\"></div></div></div><div class=\"content\"><div class=\"section\"><div class=\"section-label\">Импорт:</div><div class=\"import-row\"><input type=\"text\" class=\"import-input\" id=\"importPath\" placeholder=\"Путь к файлу...\"><button class=\"primary\" onclick=\"importConfig()\">Импорт</button></div></div><div class=\"section\"><div class=\"section-label\">Серверы:</div><div class=\"servers-list\" id=\"serversList\"><div style=\"padding:20px;text-align:center;color:#666;\">Загрузка...</div></div></div><div class=\"actions-row\"><button class=\"danger\" onclick=\"disconnect()\">Отключить</button><button onclick=\"refreshServers()\">Обновить</button><button onclick=\"deleteSelected()\">Удалить выбранный</button></div><div class=\"section\" style=\"margin-top:20px;\"><div class=\"section-label\">Лог:</div><div class=\"log-container\" id=\"logContainer\"></div></div></div><div class=\"status-bar\"><div class=\"status-bar-item\">Статус: <strong id=\"statusState\">Неизвестно</strong></div><div class=\"status-bar-item\">Сервер: <strong id=\"statusServer\">-</strong></div></div></div><script>let selectedServerId=null;function log(m){const c=document.getElementById(\"logContainer\");const now=new Date();const ts=now.toLocaleTimeString(\"ru-RU\");const e=document.createElement(\"div\");e.className=\"log-entry\";e.innerHTML=\"<span>[\"+ts+\"]</span> \"+m;c.appendChild(e);c.scrollTop=c.scrollHeight}async function fetchAPI(url,opt){opt=opt||{};try{const r=await fetch(url,Object.assign({},opt,{headers:Object.assign({\"Content-Type\":\"application/json\"},opt.headers||{})}));return await r.json()}catch(err){log(\"Ошибка: \"+err.message);throw err}}async function loadServers(){try{const servers=await fetchAPI(\"/api/servers\");const c=document.getElementById(\"serversList\");if(!servers||servers.length===0){c.innerHTML=\"<div style=\\\"padding:20px;text-align:center;color:#666;\\\">Нет серверов</div>\";return}let html=\"\";for(let i=0;i<servers.length;i++){const s=servers[i];const sel=s.id===selectedServerId?\"selected\":\"\";const chk=s.id===selectedServerId?\"checked\":\"\";html+=\"<div class=\\\"server-item \"+sel+\"\\\" onclick=\\\"selectServer('\\\"+s.id+\"')\\\"><input type=\\\"radio\\\" class=\\\"server-radio\\\" name=\\\"server\\\" \"+chk+\">\"><div class=\\\"server-info\\\"><div class=\\\"server-name\\\">\"+s.name+\":\"+s.port+\"</div><div class=\\\"server-meta\\\">\"+s.protocol+\" - \"+s.host+\"</div></div><div class=\\\"server-actions\\\"><button onclick=\\\"event.stopPropagation();connect('\\\"+s.id+\"')\\\">Connect</button></div></div>\"}c.innerHTML=html}catch(err){document.getElementById(\"serversList\").innerHTML=\"<div style=\\\"padding:20px;text-align:center;color:#f44336;\\\">Ошибка загрузки</div>\"}}function selectServer(id){selectedServerId=id;loadServers()}async function importConfig(){const path=document.getElementById(\"importPath\").value.trim();if(!path){log(\"Введите путь к файлу\");return}log(\"Импорт: \"+path);try{const result=await fetchAPI(\"/api/import\",{method:\"POST\",body:JSON.stringify({file:path})});if(result.ok){log(\"Импортировано: \"+(result.id||\"успешно\"));loadServers();document.getElementById(\"importPath\").value=\"\"}else{log(\"Ошибка импорта: \"+(result.error||\"неизвестная\"))}}catch(err){log(\"Ошибка импорта: \"+err.message)}}async function connect(id){log(\"Подключение к \"+id+\"...\");try{const result=await fetchAPI(\"/api/connect/\"+id,{method:\"POST\"});if(result.ok){log(\"Подключено к \"+id);updateStatus()}else{log(\"Ошибка: \"+(result.error||\"неизвестная\"))}}catch(err){log(\"Ошибка: \"+err.message)}}async function disconnect(){log(\"Отключение...\");try{await fetchAPI(\"/api/disconnect\",{method:\"POST\"});log(\"Отключено\");updateStatus()}catch(err){log(\"Ошибка: \"+err.message)}}async function deleteSelected(){if(!selectedServerId){log(\"Выберите сервер\");return}log(\"Удаление \"+selectedServerId+\"...\");try{await fetchAPI(\"/api/delete/\"+selectedServerId,{method:\"DELETE\"});log(\"Сервер удалён\");selectedServerId=null;loadServers()}catch(err){log(\"Ошибка: \"+err.message)}}function refreshServers(){log(\"Обновление...\");loadServers();updateStatus()}async function updateStatus(){try{const st=await fetchAPI(\"/api/status\");document.getElementById(\"statusState\").textContent=st.state||\"unknown\";document.getElementById(\"statusServer\").textContent=st.serverId||\"-\";const hs=document.getElementById(\"headerStatus\");const hd=document.getElementById(\"headerDot\");if(st.state===\"connected\")hs.textContent=\"Подключено\";else if(st.state===\"connecting\")hs.textContent=\"Подключение...\";else if(st.state===\"error\")hs.textContent=\"Ошибка\";else hs.textContent=\"Отключено\";hd.className=\"status-dot \"+(st.state||\"disconnected\")}catch(err){console.error(err)}}log(\"TunnelCraft Control запущен\");loadServers();updateStatus();setInterval(updateStatus,5000)</script></body></html>"
