package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	pb "github.com/ChickenRamen500/tunnelcraft/core/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run ./cmd/test-connect/ <server-uuid>")
	}
	serverID := os.Args[1]

	conn, err := grpc.Dial(
		"127.0.0.1:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()

	ts := pb.NewTunnelServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Connecting to %s...\n", serverID)
	resp, err := ts.Connect(ctx, &pb.ConnectRequest{
		ServerId: serverID,
	})
	if err != nil {
		log.Fatal("Connect failed:", err)
	}

	fmt.Println("============================================")
	fmt.Println("✅ Connected!")
	fmt.Println("============================================")
	fmt.Printf("  State:     %v\n", resp.State)
	fmt.Printf("  ServerId:  %s\n", resp.ServerId)
	fmt.Printf("  SocksPort: %d\n", resp.SocksPort)
	fmt.Printf("  HttpPort:  %d\n", resp.HttpPort)
	if resp.Error != "" {
		fmt.Printf("  Error:     %s\n", resp.Error)
	}
	fmt.Println("============================================")
}