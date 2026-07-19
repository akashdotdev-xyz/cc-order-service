package main

import (
	"database/sql"
	"log"
	"net/http"

	_ "github.com/lib/pq"

	"order-service/config"
	"order-service/internal/api/handlers"
	"order-service/internal/api/routes"
	"order-service/internal/queue/kafka"
	"order-service/internal/repository/postgres"
	"order-service/internal/service"
)

func main() {
	cfg := config.Load()

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}
	defer db.Close()

	orderRepo := postgres.NewOrderRepository(db)
	// LogWriter stands in for a real Kafka client until one is wired up.
	orderPublisher := kafka.NewPublisher(&kafka.LogWriter{Topic: "order.created"}, "order.created")
	orderService := service.NewOrderService(orderRepo, orderPublisher)
	orderHandler := handlers.NewOrderHandler(orderService)

	router := routes.NewRouter(orderHandler)

	log.Printf("order-service listening on %s", cfg.Addr())
	if err := http.ListenAndServe(cfg.Addr(), router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
