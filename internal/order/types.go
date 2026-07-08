// Package order owns placed orders: creation (called from checkout inside
// its transaction), status transitions, and customer-facing lookup by
// order number + email.
package order

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("order: not found")

type Status string

const (
	StatusPending   Status = "pending"
	StatusPaid      Status = "paid"
	StatusFulfilled Status = "fulfilled"
	StatusShipped   Status = "shipped"
	StatusDelivered Status = "delivered"
	StatusCancelled Status = "cancelled"
)

type Address struct {
	Name       string
	Line1      string
	City       string
	PostalCode string
	Country    string
}

type Item struct {
	VariantID      int64
	ProductTitle   string
	VariantName    string
	UnitPriceCents int64
	Qty            int
}

func (i Item) LineTotalCents() int64 { return i.UnitPriceCents * int64(i.Qty) }

type Order struct {
	ID               int64
	OrderNumber      string
	Email            string
	Status           Status
	SubtotalCents    int64
	ShippingCents    int64
	DiscountCents    int64
	DiscountCode     string
	TotalCents       int64
	Shipping         Address
	PaymentReference string
	TrackingNumber   string
	Items            []Item
	CreatedAt        time.Time
}

// OrderSummary is what a list view needs — no line items.
type OrderSummary struct {
	ID          int64
	OrderNumber string
	Email       string
	Status      Status
	TotalCents  int64
	CreatedAt   time.Time
}

var ErrInvalidTransition = errors.New("order: invalid status transition")

// validNextStatuses defines the fulfillment pipeline. Cancellation is only
// allowed before an order ships — once it's shipped, cancelling doesn't
// undo a physical parcel in transit.
var validNextStatuses = map[Status][]Status{
	StatusPending:   {StatusPaid, StatusCancelled},
	StatusPaid:      {StatusFulfilled, StatusCancelled},
	StatusFulfilled: {StatusShipped, StatusCancelled},
	StatusShipped:   {StatusDelivered},
	StatusDelivered: {},
	StatusCancelled: {},
}

func CanTransition(from, to Status) bool {
	for _, s := range validNextStatuses[from] {
		if s == to {
			return true
		}
	}
	return false
}
