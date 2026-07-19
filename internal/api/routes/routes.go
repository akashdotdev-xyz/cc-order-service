package routes

import (
	"net/http"

	"order-service/internal/api/handlers"
	"order-service/internal/api/middlewares"
)

// NewRouter wires up all HTTP routes for the service.
func NewRouter(orderHandler *handlers.OrderHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /orders", orderHandler.ListOrders)
	mux.HandleFunc("POST /orders", orderHandler.CreateOrder)

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	return middlewares.Logging(mux)
}
