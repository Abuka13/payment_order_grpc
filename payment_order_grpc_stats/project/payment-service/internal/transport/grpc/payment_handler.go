package grpc

import (
	"context"
	"payment-service/internal/usecase"
	"payment-service/pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PaymentGRPCHandler struct {
	pb.UnimplementedPaymentServiceServer
	uc *usecase.PaymentUsecase
}

func NewPaymentGRPCHandler(uc *usecase.PaymentUsecase) *PaymentGRPCHandler {
	return &PaymentGRPCHandler{
		uc: uc,
	}
}

func (h *PaymentGRPCHandler) ProcessPayment(ctx context.Context, req *pb.PaymentRequest) (*pb.PaymentResponse, error) {
	if req.OrderId == "" || req.Amount <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order_id or amount")
	}

	payment, err := h.uc.Create(req.OrderId, req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to process payment: %v", err)
	}

	return &pb.PaymentResponse{
		Id:            payment.ID,
		OrderId:       payment.OrderID,
		TransactionId: payment.TransactionID,
		Amount:        payment.Amount,
		Status:        payment.Status,
		CreatedAt:     timestamppb.Now(),
	}, nil
}

func (h *PaymentGRPCHandler) GetPaymentByOrderID(ctx context.Context, req *pb.GetPaymentRequest) (*pb.PaymentResponse, error) {
	if req.OrderId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "order_id is required")
	}

	payment, err := h.uc.GetByOrderID(req.OrderId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "payment not found: %v", err)
	}

	return &pb.PaymentResponse{
		Id:            payment.ID,
		OrderId:       payment.OrderID,
		TransactionId: payment.TransactionID,
		Amount:        payment.Amount,
		Status:        payment.Status,
		CreatedAt:     timestamppb.Now(),
	}, nil
}

func (h *PaymentGRPCHandler) GetPaymentStats(ctx context.Context, req *pb.GetPaymentStatsRequest) (*pb.PaymentStats, error) {
	stats, err := h.uc.GetStats()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get payment stats: %v", err)
	}

	return &pb.PaymentStats{
		TotalCount:      stats.TotalCount,
		AuthorizedCount: stats.AuthorizedCount,
		DeclinedCount:   stats.DeclinedCount,
		TotalAmount:     stats.TotalAmount,
	}, nil
}
