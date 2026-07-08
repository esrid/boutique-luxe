// Package checkout turns a cart into a paid order: re-validate stock,
// decrement it, record the order, charge payment — atomically.
package checkout

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/esrid/maison/internal/cart"
	"github.com/esrid/maison/internal/discount"
	"github.com/esrid/maison/internal/order"
	"github.com/esrid/maison/internal/payment"
)

var (
	ErrEmptyCart       = errors.New("checkout: cart is empty")
	ErrOutOfStock      = errors.New("checkout: an item is no longer available in the requested quantity")
	ErrInvalidDiscount = errors.New("checkout: discount code is invalid or expired")
)

type Input struct {
	Email        string
	Name         string
	Address      string
	City         string
	PostalCode   string
	Country      string
	DiscountCode string // "" means none
}

type Service struct {
	db        *sql.DB
	carts     *cart.Store
	orders    *order.Store
	discounts *discount.Store
	payment   payment.Provider
}

func New(db *sql.DB, carts *cart.Store, orders *order.Store, discounts *discount.Store, prov payment.Provider) *Service {
	return &Service{db: db, carts: carts, orders: orders, discounts: discounts, payment: prov}
}

// PlaceOrder re-validates stock, decrements it, creates the order, and
// charges payment, all inside one transaction: if the charge "fails" the
// stock decrement rolls back with it, so a failed payment never leaves
// inventory short.
//
// ponytail: correctness here isn't from row-level locking — internal/db
// caps the sqlite connection pool at 1 (see db.Open), so only one write
// transaction runs at a time app-wide, which already rules out concurrent
// overselling. If this ever moves to a multi-writer DB, add row locking
// (`SELECT ... FOR UPDATE`) on the variant rows before the stock check.
func (s *Service) PlaceOrder(ctx context.Context, cartID int64, in Input) (*order.Order, error) {
	c, err := s.carts.Load(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("load cart: %w", err)
	}
	if len(c.Items) == 0 {
		return nil, ErrEmptyCart
	}

	var discountCents int64
	var discountCode string
	if in.DiscountCode != "" {
		code, err := s.discounts.FindActiveByCode(ctx, in.DiscountCode)
		if err != nil {
			return nil, ErrInvalidDiscount
		}
		discountCents = code.ComputeDiscountCents(c.SubtotalCents)
		discountCode = code.Code
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	items := make([]order.Item, 0, len(c.Items))
	for _, ci := range c.Items {
		var stock int
		if err := tx.QueryRowContext(ctx, `SELECT stock_qty FROM product_variants WHERE id = ?`, ci.VariantID).Scan(&stock); err != nil {
			return nil, fmt.Errorf("check stock: %w", err)
		}
		if stock < ci.Qty {
			return nil, fmt.Errorf("%w: %s (%s)", ErrOutOfStock, ci.ProductTitle, ci.VariantName)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE product_variants SET stock_qty = stock_qty - ? WHERE id = ?`, ci.Qty, ci.VariantID); err != nil {
			return nil, fmt.Errorf("decrement stock: %w", err)
		}
		items = append(items, order.Item{
			VariantID:      ci.VariantID,
			ProductTitle:   ci.ProductTitle,
			VariantName:    ci.VariantName,
			UnitPriceCents: ci.UnitPriceCents,
			Qty:            ci.Qty,
		})
	}

	orderNumber, err := order.NewOrderNumber()
	if err != nil {
		return nil, fmt.Errorf("generate order number: %w", err)
	}

	totalCents := c.SubtotalCents - discountCents

	orderID, err := s.orders.Create(ctx, tx, order.CreateParams{
		OrderNumber:   orderNumber,
		Email:         in.Email,
		SubtotalCents: c.SubtotalCents,
		DiscountCents: discountCents,
		DiscountCode:  discountCode,
		TotalCents:    totalCents,
		Shipping: order.Address{
			Name: in.Name, Line1: in.Address, City: in.City,
			PostalCode: in.PostalCode, Country: in.Country,
		},
		Items: items,
	})
	if err != nil {
		return nil, err
	}

	receipt, err := s.payment.Charge(ctx, totalCents, orderNumber)
	if err != nil {
		return nil, fmt.Errorf("charge payment: %w", err)
	}
	if err := s.orders.MarkPaid(ctx, tx, orderID, receipt.Reference); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit order: %w", err)
	}

	// Best-effort: a cart line surviving a committed order is cosmetic
	// clutter, not lost data, so a failure here isn't worth failing the
	// checkout that already succeeded.
	for _, ci := range c.Items {
		_ = s.carts.RemoveItem(ctx, cartID, ci.VariantID)
	}

	return s.orders.GetByNumberAndEmail(ctx, orderNumber, in.Email)
}
