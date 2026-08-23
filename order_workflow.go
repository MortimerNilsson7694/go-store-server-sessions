package main

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type orderStatus string

const (
	statusPaid      orderStatus = "paid"
	statusFulfilled orderStatus = "fulfilled"
)

type order struct {
	ID        string      `json:"id"`
	UserID    string      `json:"user_id"`
	SKU       string      `json:"sku"`
	Quantity  int         `json:"quantity"`
	Status    orderStatus `json:"status"`
	ReceiptID string      `json:"receipt_id,omitempty"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type orderBook struct {
	mu     sync.Mutex
	orders map[string]order
}

func newOrderBook() *orderBook {
	return &orderBook{orders: make(map[string]order)}
}

func (b *orderBook) checkout(id, userID, sku string, quantity int, now time.Time) (order, error) {
	if id == "" || userID == "" || sku == "" || quantity < 1 {
		return order{}, errors.New("order_id, sku, and positive quantity are required")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if existing, ok := b.orders[id]; ok {
		if existing.UserID != userID || existing.SKU != sku || existing.Quantity != quantity {
			return order{}, errors.New("order_id already names another checkout")
		}
		return existing, nil
	}
	created := order{ID: id, UserID: userID, SKU: sku, Quantity: quantity, Status: statusPaid, UpdatedAt: now}
	b.orders[id] = created
	return created, nil
}

func (b *orderBook) fulfill(id, userID string, now time.Time) (order, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	current, ok := b.orders[id]
	if !ok || current.UserID != userID {
		return order{}, errors.New("order not found")
	}
	if current.Status == statusFulfilled {
		return current, nil
	}
	if current.Status != statusPaid {
		return order{}, errors.New("order is not paid")
	}
	current.Status = statusFulfilled
	current.ReceiptID = fmt.Sprintf("receipt-%s", current.ID)
	current.UpdatedAt = now
	b.orders[id] = current
	return current, nil
}

func (b *orderBook) list(userID string) []order {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := make([]order, 0)
	for _, item := range b.orders {
		if item.UserID == userID {
			result = append(result, item)
		}
	}
	return result
}
