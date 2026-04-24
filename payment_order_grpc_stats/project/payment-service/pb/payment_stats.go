package pb

// GetPaymentStatsRequest is the request message for GetPaymentStats RPC.
type GetPaymentStatsRequest struct{}

// PaymentStats is the response message for GetPaymentStats RPC.
type PaymentStats struct {
	TotalCount      int64 `protobuf:"varint,1,opt,name=total_count,json=totalCount,proto3" json:"total_count,omitempty"`
	AuthorizedCount int64 `protobuf:"varint,2,opt,name=authorized_count,json=authorizedCount,proto3" json:"authorized_count,omitempty"`
	DeclinedCount   int64 `protobuf:"varint,3,opt,name=declined_count,json=declinedCount,proto3" json:"declined_count,omitempty"`
	TotalAmount     int64 `protobuf:"varint,4,opt,name=total_amount,json=totalAmount,proto3" json:"total_amount,omitempty"` // сумма всех amount в cents
}

func (x *GetPaymentStatsRequest) Reset()         {}
func (x *GetPaymentStatsRequest) String() string  { return "GetPaymentStatsRequest{}" }
func (x *GetPaymentStatsRequest) ProtoMessage()   {}

func (x *PaymentStats) Reset()         {}
func (x *PaymentStats) String() string { return "PaymentStats{}" }
func (x *PaymentStats) ProtoMessage()  {}

func (x *PaymentStats) GetTotalCount() int64 {
	if x != nil {
		return x.TotalCount
	}
	return 0
}

func (x *PaymentStats) GetAuthorizedCount() int64 {
	if x != nil {
		return x.AuthorizedCount
	}
	return 0
}

func (x *PaymentStats) GetDeclinedCount() int64 {
	if x != nil {
		return x.DeclinedCount
	}
	return 0
}

func (x *PaymentStats) GetTotalAmount() int64 {
	if x != nil {
		return x.TotalAmount
	}
	return 0
}
