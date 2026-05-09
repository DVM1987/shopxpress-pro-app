package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func MustOpen(dsn string) *sql.DB {
	pool, err := sql.Open("pgx", dsn)
	if err != nil {
		panic(fmt.Errorf("open db: %w", err))
	}
	pool.SetMaxOpenConns(20)
	pool.SetMaxIdleConns(5)
	pool.SetConnMaxLifetime(30 * time.Minute)
	pool.SetConnMaxIdleTime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.PingContext(ctx); err != nil {
		panic(fmt.Errorf("ping db: %w", err))
	}
	return pool
}
