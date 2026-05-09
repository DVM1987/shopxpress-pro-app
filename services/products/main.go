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

type Product struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
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
	mux.HandleFunc("GET /products", listProducts)
	mux.HandleFunc("GET /products/{id}", getProduct)

	addr := ":" + srv.Env("PORT", "8081")
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

func listProducts(w http.ResponseWriter, r *http.Request) {
	ps, err := repo.List(r.Context())
	if err != nil {
		slog.Error("list products", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, ps)
}

func getProduct(w http.ResponseWriter, r *http.Request) {
	p, err := repo.GetByID(r.Context(), r.PathValue("id"))
	if errors.Is(err, ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		return
	}
	if err != nil {
		slog.Error("get product", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
