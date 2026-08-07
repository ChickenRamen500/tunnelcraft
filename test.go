package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/ChickenRamen500/tunnelcraft/proto/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient(
		"127.0.0.1:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()

	// 1. Import
	svc := pb.NewServerServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	content := "[Interface]\nPrivateKey = mPo2saI7WgiQXf5TEQ7zL50A7C8oNWP2jV4h4f8HpFI=\nAddress = 10.146.1.57/32\nDNS = 1.1.1.1, 8.8.4.4\nMTU = 1380\n\n[Peer]\nPublicKey = p0ix2urEpkj97oYak/YMv8GYa14fNydAlaePAtF0vw0=\nPresharedKey = MZc/cb7FjIqW8/RFWlw9ZwcrNlUq4893TQ2ak0wHMUE=\nAllowedIPs = 0.0.0.0/0, ::/0\nPersistentKeepalive = 25\nEndpoint = arg01wg.kcufwfgnkr.net:50146"

	resp, err := svc.ImportServers(ctx, &pb.ImportServersRequest{Content: content})
	if err != nil {
		log.Fatal("ImportServers:", err)
	}
	fmt.Printf("✅ Imported!\n")
	fmt.Printf("   IDs: %v\n", resp.ImportedServerIds)
	fmt.Printf("   Parsed: %d, Imported: %d\n", resp.TotalParsed, resp.TotalImported)
	if len(resp.Errors) > 0 {
		fmt.Printf("   Errors: %v\n", resp.Errors)
	}
}