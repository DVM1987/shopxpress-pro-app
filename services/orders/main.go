package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/DVM1987/shopxpress-pro-app/pkg/db"
	"github.com/DVM1987/shopxpress-pro-app/pkg/srv"
)

type OrderItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type Order struct {
	ID        string      `json:"id"`
	Items     []OrderItem `json:"items"`
	CreatedAt time.Time   `json:"created_at"`
}

var repo *Repo

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	pool := db.MustOpen(srv.MustEnv("DATABASE_URL"))
	defer pool.Close()
	repo = NewRepo(pool)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", srv.Health)
	mux.HandleFunc("GET /readyz", readyz(pool))
	mux.HandleFunc("POST /orders", createOrder)
	mux.HandleFunc("GET /orders/{id}", getOrder)

	addr := ":" + srv.Env("PORT", "8082")
	srv.Run(srv.New(addr, srv.LogRequest(mux)))
}

func readyz(pool *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer cancel()
		if err := pool.PingContext(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "db unreachable"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func createOrder(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Items []OrderItem `json:"items"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if len(in.Items) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "items required"})
		return
	}
	for _, it := range in.Items {
		if it.ProductID == "" || it.Quantity <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid item"})
			return
		}
	}

	o, err := repo.Create(r.Context(), in.Items)
	if err != nil {
		slog.Error("create order", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusCreated, o)
}

func getOrder(w http.ResponseWriter, r *http.Request) {
	o, err := repo.GetByID(r.Context(), r.PathValue("id"))
	if errors.Is(err, ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order not found"})
		return
	}
	if err != nil {
		slog.Error("get order", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
