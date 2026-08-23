package main

import (
	"testing"
	"time"
)

func TestFulfillmentCreatesReceiptForPaidOrder(t *testing.T) {
	fixed := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		checkout    bool
		wantStatus  orderStatus
		wantReceipt string
		wantErr     bool
	}{
		{name: "paid checkout is fulfilled", checkout: true, wantStatus: statusFulfilled, wantReceipt: "receipt-order-42"},
		{name: "unknown order is rejected", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			book := newOrderBook()
			if tt.checkout {
				if _, err := book.checkout("order-42", "user-7", "mug-black", 2, fixed); err != nil {
					t.Fatal(err)
				}
			}
			got, err := book.fulfill("order-42", "user-7", fixed.Add(time.Minute))
			if (err != nil) != tt.wantErr {
				t.Fatalf("fulfill error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got.Status != tt.wantStatus || got.ReceiptID != tt.wantReceipt {
				t.Fatalf("got status %q receipt %q", got.Status, got.ReceiptID)
			}
		})
	}
}
