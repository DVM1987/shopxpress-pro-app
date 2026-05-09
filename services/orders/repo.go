package main

import (
	"context"
	"database/sql"
	"errors"
)

var ErrNotFound = errors.New("not found")

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

func (r *Repo) Create(ctx context.Context, items []OrderItem) (Order, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Order{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var o Order
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO orders DEFAULT VALUES
		RETURNING id, created_at`).
		Scan(&o.ID, &o.CreatedAt); err != nil {
		return Order{}, err
	}

	for _, it := range items {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO order_items (order_id, product_id, quantity)
			VALUES ($1, $2, $3)`,
			o.ID, it.ProductID, it.Quantity); err != nil {
			return Order{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Order{}, err
	}
	o.Items = items
	return o, nil
}

func (r *Repo) GetByID(ctx context.Context, id string) (Order, error) {
	var o Order
	err := r.db.QueryRowContext(ctx, `
		SELECT id, created_at
		FROM orders
		WHERE id = $1`, id).
		Scan(&o.ID, &o.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT product_id, quantity
		FROM order_items
		WHERE order_id = $1
		ORDER BY id ASC`, id)
	if err != nil {
		return Order{}, err
	}
	defer rows.Close()

	o.Items = make([]OrderItem, 0)
	for rows.Next() {
		var it OrderItem
		if err := rows.Scan(&it.ProductID, &it.Quantity); err != nil {
			return Order{}, err
		}
		o.Items = append(o.Items, it)
	}
	return o, rows.Err()
}
