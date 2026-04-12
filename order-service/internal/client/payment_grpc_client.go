package client

import (
	"context"
	"order-service/pb"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type PaymentGRPCClient struct {
	conn   *grpc.ClientConn
	client pb.PaymentServiceClient
}

func NewPaymentGRPCClient(addr string) (*PaymentGRPCClient, error) {
	conn, err := grpc.Dial(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(4194304)),
	)
	if err != nil {
		return nil, err
	}

	return &PaymentGRPCClient{
		conn:   conn,
		client: pb.NewPaymentServiceClient(conn),
	}, nil
}

func (p *PaymentGRPCClient) CreatePayment(orderID string, amount int64) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := p.client.ProcessPayment(ctx, &pb.PaymentRequest{
		OrderId: orderID,
		Amount:  amount,
	})

	if err != nil {
		if status.Code(err) == codes.Unavailable {
			return "", "", grpc.ErrClientConnClosing
		}
		return "", "", err
	}

	return resp.Status, resp.TransactionId, nil
}

func (p *PaymentGRPCClient) GetPayment(orderID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := p.client.GetPaymentByOrderID(ctx, &pb.GetPaymentRequest{
		OrderId: orderID,
	})

	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", grpc.ErrClientConnClosing
		}
		return "", err
	}

	return resp.Status, nil
}

func (p *PaymentGRPCClient) Close() error {
	return p.conn.Close()
}

