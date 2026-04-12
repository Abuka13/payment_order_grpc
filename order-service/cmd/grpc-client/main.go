package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"order-service/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	orderID := flag.String("order-id", "", "Order ID to subscribe to")
	serverAddr := flag.String("addr", "localhost:9090", "Server address")
	flag.Parse()

	if *orderID == "" {
		log.Fatal("order-id flag is required")
	}

	conn, err := grpc.Dial(
		*serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	client := pb.NewOrderServiceClient(conn)

	ctx := context.Background()
	stream, err := client.SubscribeToOrderUpdates(ctx, &pb.OrderRequest{
		OrderId: *orderID,
	})
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	fmt.Printf("Subscribed to order %s. Waiting for updates...\n", *orderID)

	for {
		update, err := stream.Recv()
		if err != nil {
			log.Fatalf("Error receiving update: %v", err)
		}

		fmt.Printf("[%s] Order Status: %s - %s\n",
			update.UpdatedAt.AsTime().Format("15:04:05"),
			update.Status,
			update.Message,
		)
	}
}

