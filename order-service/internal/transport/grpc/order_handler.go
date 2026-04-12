package grpc

import (
	"context"
	"database/sql"
	"log"
	"order-service/internal/repository"
	"order-service/internal/usecase"
	"order-service/pb"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrderGRPCHandler struct {
	pb.UnimplementedOrderServiceServer
	uc   *usecase.OrderUsecase
	repo repository.OrderRepository
}

func NewOrderGRPCHandler(uc *usecase.OrderUsecase, repo repository.OrderRepository) *OrderGRPCHandler {
	return &OrderGRPCHandler{
		uc:   uc,
		repo: repo,
	}
}

// SubscribeToOrderUpdates implements server-side streaming RPC for real-time order updates
func (h *OrderGRPCHandler) SubscribeToOrderUpdates(req *pb.OrderRequest, stream pb.OrderService_SubscribeToOrderUpdatesServer) error {
	if req.OrderId == "" {
		return status.Errorf(codes.InvalidArgument, "order_id is required")
	}

	// Verify order exists
	order, err := h.uc.GetByID(req.OrderId)
	if err != nil {
		if err == sql.ErrNoRows {
			return status.Errorf(codes.NotFound, "order not found")
		}
		return status.Errorf(codes.Internal, "failed to get order: %v", err)
	}

	log.Printf("[gRPC Streaming] Client subscribed to order %s (initial status: %s)\n", req.OrderId, order.Status)

	// Send initial status
	initialUpdate := &pb.OrderStatusUpdate{
		OrderId:   req.OrderId,
		Status:    order.Status,
		UpdatedAt: timestamppb.New(order.CreatedAt),
		Message:   "Subscribed to order updates",
	}

	if err := stream.Send(initialUpdate); err != nil {
		log.Printf("[gRPC Streaming] Error sending initial update: %v\n", err)
		return err
	}

	// Poll database for updates every 2 seconds
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	lastStatus := order.Status
	timeoutContext, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for {
		select {
		case <-timeoutContext.Done():
			log.Printf("[gRPC Streaming] Client unsubscribed from order %s (timeout)\n", req.OrderId)
			return nil
		case <-stream.Context().Done():
			log.Printf("[gRPC Streaming] Client unsubscribed from order %s (context done)\n", req.OrderId)
			return nil
		case <-ticker.C:
			// Check for order status updates in database
			updatedOrder, err := h.repo.GetByID(req.OrderId)
			if err != nil {
				if err == sql.ErrNoRows {
					return status.Errorf(codes.NotFound, "order was deleted")
				}
				log.Printf("[gRPC Streaming] Error fetching updated order: %v\n", err)
				continue
			}

			// If status changed, send update
			if updatedOrder.Status != lastStatus {
				log.Printf("[gRPC Streaming] Order %s status changed from %s to %s\n", req.OrderId, lastStatus, updatedOrder.Status)

				statusUpdate := &pb.OrderStatusUpdate{
					OrderId:   req.OrderId,
					Status:    updatedOrder.Status,
					UpdatedAt: timestamppb.New(time.Now()),
					Message:   "Order status changed to " + updatedOrder.Status,
				}

				if err := stream.Send(statusUpdate); err != nil {
					log.Printf("[gRPC Streaming] Error sending update: %v\n", err)
					return err
				}

				lastStatus = updatedOrder.Status
			}
		}
	}
}

