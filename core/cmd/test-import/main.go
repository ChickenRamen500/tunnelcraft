package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/ChickenRamen500/tunnelcraft/core/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.Dial(
		"127.0.0.1:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()

	// WireGuard .conf content
	content := `[Interface]
PrivateKey = mPo2saI7WgiQXf5TEQ7zL50A7C8oNWP2jV4h4f8HpFI=
Address = 10.146.1.57/32
DNS = 1.1.1.1, 8.8.4.4
MTU = 1380

[Peer]
PublicKey = p0ix2urEpkj97oYak/YMv8GYa14fNydAlaePAtF0vw0=
PresharedKey = MZc/cb7FjIqW8/RFWlw9ZwcrNlUq4893TQ2ak0wHMUE=
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25
Endpoint = arg01wg.kcufwfgnkr.net:50146`

	svc := pb.NewServerServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := svc.ImportServers(ctx, &pb.ImportServersRequest{
		Content: content,
	})
	if err != nil {
		log.Fatal("ImportServers failed:", err)
	}

	fmt.Println("============================================")
	fmt.Println("✅ Import successful!")
	fmt.Println("============================================")
	for i, id := range resp.ImportedServerIds {
		fmt.Printf("  [%d] %s\n", i+1, id)
	}
	fmt.Printf("  Total parsed:   %d\n", resp.TotalParsed)
	fmt.Printf("  Total imported: %d\n", resp.TotalImported)
	if len(resp.Errors) > 0 {
		fmt.Printf("  Errors: %v\n", resp.Errors)
	}
	fmt.Println("============================================")
}