// Package main provides the tcctl CLI utility for managing TunnelCraft daemon.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ChickenRamen500/tunnelcraft/core/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const defaultAddr = "127.0.0.1:50051"

var stateNames = map[int32]string{
	0: "unspecified",
	1: "disconnected",
	2: "connecting",
	3: "connected",
	4: "disconnecting",
	5: "error",
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	conn, err := grpc.Dial(defaultAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot connect to daemon. Is tunnelcraftd running?\n")
		os.Exit(1)
	}
	defer conn.Close()

	tunnelClient := proto.NewTunnelServiceClient(conn)
	serverClient := proto.NewServerServiceClient(conn)

	ctx := context.Background()

	switch cmd {
	case "import":
		if len(args) < 1 {
			fmt.Fprintf(os.Stderr, "Usage: tcctl import <file.conf>\n")
			os.Exit(1)
		}
		importConfig(ctx, serverClient, args[0])

	case "import-clip":
		importFromClipboard(ctx, serverClient)

	case "list":
		listServers(ctx, serverClient)

	case "connect":
		if len(args) < 1 {
			fmt.Fprintf(os.Stderr, "Usage: tcctl connect <id>\n")
			os.Exit(1)
		}
		connect(ctx, tunnelClient, args[0])

	case "disconnect":
		disconnect(ctx, tunnelClient)

	case "status":
		showStatus(ctx, tunnelClient)

	case "delete":
		if len(args) < 1 {
			fmt.Fprintf(os.Stderr, "Usage: tcctl delete <id>\n")
			os.Exit(1)
		}
		deleteServer(ctx, serverClient, args[0])

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`tcctl - TunnelCraft CLI

Usage:
  tcctl import <file.conf>     Import config from file
  tcctl import-clip            Import from clipboard
  tcctl list                   List servers
  tcctl connect <id>           Connect to server
  tcctl disconnect             Disconnect
  tcctl status                 Show connection status
  tcctl delete <id>            Delete server`)
}

func importConfig(ctx context.Context, client proto.ServerServiceClient, filePath string) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error reading file: %v\n", err)
		os.Exit(1)
	}

	resp, err := client.ImportServers(ctx, &proto.ImportServersRequest{
		Content: string(data),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}

	if len(resp.ImportedServerIds) > 0 {
		fmt.Printf("✅ Imported: %s\n", resp.ImportedServerIds[0])
	} else {
		fmt.Println("✅ Imported successfully")
	}
}

func importFromClipboard(ctx context.Context, client proto.ServerServiceClient) {
	cmd := exec.Command("powershell", "-Command", "Get-Clipboard")
	output, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error reading clipboard: %v\n", err)
		os.Exit(1)
	}

	resp, err := client.ImportServers(ctx, &proto.ImportServersRequest{
		Content: strings.TrimSpace(string(output)),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}

	if len(resp.ImportedServerIds) > 0 {
		fmt.Printf("✅ Imported: %s\n", resp.ImportedServerIds[0])
	} else {
		fmt.Println("✅ Imported successfully")
	}
}

func listServers(ctx context.Context, client proto.ServerServiceClient) {
	resp, err := client.ListServers(ctx, &proto.ListServersRequest{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%-40s %-12s %s\n", "ID", "PROTOCOL", "HOST:PORT")
	for _, s := range resp.Servers {
		protocol := protocolName(s.Protocol)
		hostPort := fmt.Sprintf("%s:%d", s.Host, s.Port)
		fmt.Printf("%-40s %-12s %s\n", truncateID(s.Id), protocol, hostPort)
	}
}

func connect(ctx context.Context, client proto.TunnelServiceClient, serverID string) {
	_, err := client.Connect(ctx, &proto.ConnectRequest{ServerId: serverID})
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Connected to %s\n", serverID)
}

func disconnect(ctx context.Context, client proto.TunnelServiceClient) {
	_, err := client.Disconnect(ctx, &proto.DisconnectRequest{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Disconnected")
}

func showStatus(ctx context.Context, client proto.TunnelServiceClient) {
	resp, err := client.GetConnectionStatus(ctx, &proto.GetConnectionStatusRequest{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}

	state := stateNames[int32(resp.State)]
	if resp.State == 3 { // connected
		duration := ""
		if resp.ConnectedAt != nil {
			duration = fmt.Sprintf(" | Connected at: %s", resp.ConnectedAt.AsTime().Format("15:04:05"))
		}
		fmt.Printf("State: %s | Server: %s%s\n", state, truncateID(resp.ServerId), duration)
	} else {
		fmt.Printf("Disconnected\n")
	}
}

func deleteServer(ctx context.Context, client proto.ServerServiceClient, serverID string) {
	_, err := client.DeleteServer(ctx, &proto.DeleteServerRequest{Id: serverID})
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Deleted")
}

func protocolName(p proto.Protocol) string {
	switch p {
	case proto.Protocol_PROTOCOL_VLESS:
		return "vless"
	case proto.Protocol_PROTOCOL_VMESS:
		return "vmess"
	case proto.Protocol_PROTOCOL_WIREGUARD:
		return "wireguard"
	case proto.Protocol_PROTOCOL_HYSTERIA:
		return "hysteria"
	case proto.Protocol_PROTOCOL_AMNEZIAWG:
		return "amnezia-wg"
	case proto.Protocol_PROTOCOL_SHADOWSOCKS:
		return "shadowsocks"
	default:
		return "unknown"
	}
}

func truncateID(id string) string {
	if len(id) > 36 {
		return id[:36]
	}
	return id
}
