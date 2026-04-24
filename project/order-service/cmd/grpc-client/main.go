package main

import (
	"context"
	"flag"
	"io"
	"log"
	"order-service/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	addr := flag.String("addr", "localhost:9090", "Order gRPC server address")
	orderID := flag.String("order", "", "Order ID to subscribe to")
	flag.Parse()

	if *orderID == "" {
		log.Fatal("Please provide --order <order_id>")
	}

	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewOrderServiceClient(conn)

	stream, err := client.SubscribeToOrderUpdates(context.Background(), &pb.OrderRequest{
		OrderId: *orderID,
	})
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	log.Printf("Subscribed to order %s updates...\n", *orderID)

	for {
		update, err := stream.Recv()
		if err == io.EOF {
			log.Println("Stream closed by server")
			break
		}
		if err != nil {
			log.Fatalf("Error receiving update: %v", err)
		}
		log.Printf("[UPDATE] Order: %s | Status: %s | Message: %s\n",
			update.OrderId, update.Status, update.Message)
	}
}
