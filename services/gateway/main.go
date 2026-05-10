package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/DVM1987/shopxpress-pro-app/pkg/srv"
)

var (
	productsURL = srv.Env("PRODUCTS_URL", "http://localhost:8081")
	ordersURL   = srv.Env("ORDERS_URL", "http://localhost:8082")
	httpClient  = &http.Client{Timeout: 3 * time.Second}
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", srv.Health)
	mux.HandleFunc("GET /readyz", srv.Health)
	mux.HandleFunc("GET /api/products", proxyGet(productsURL+"/products"))
	mux.HandleFunc("POST /api/orders", forwardOrder)
	mux.HandleFunc("GET /api/orders/{id}", forwardGetOrder)

	addr := ":" + srv.Env("PORT", "8080")
	srv.Run(srv.New(addr, srv.LogRequest(mux)))
}

func proxyGet(target string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			slog.Error("upstream call failed", "target", target, "err", err)
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "upstream unavailable"})
			return
		}
		defer resp.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}
}

func forwardOrder(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, ordersURL+"/orders", bytes.NewReader(body))
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		slog.Error("orders upstream failed", "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "orders unavailable"})
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func forwardGetOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, ordersURL+"/orders/"+id, nil)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		slog.Error("orders upstream failed", "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "orders unavailable"})
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// CI test: trigger bot bump tag e2e (Sub-comp 0.7.8)
