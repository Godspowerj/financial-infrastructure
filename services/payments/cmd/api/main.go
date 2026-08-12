package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver with database/sql

	payments "fintech-lab/services/payments/internal"
)

// db is a package-level connection *pool*, not a single connection.
var db *sql.DB

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Sensible local default matching docker-compose.yml values.
		dsn = "postgres://fintech:fintech@localhost:5432/payments?sslmode=disable"
	}

	var err error
	db, err = sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("failed to configure database: %v", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	bankURL := os.Getenv("BANK_SERVICE_URL")
	if bankURL == "" {
		bankURL = "http://localhost:8081"
	}


	// Wire up layers: Repository + Gateway -> Service -> Handler
	repo := payments.NewRepository(db)
	gateway := payments.NewHTTPBankGateway(bankURL)
	service := payments.NewService(repo, gateway)
	handler := payments.NewHandler(service)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /webhooks/bank", handler.HandleBankWebhook)
	mux.HandleFunc("POST /payments", handler.CreatePayment)
	mux.HandleFunc("GET /payments/{id}", handler.GetPayment)
	mux.HandleFunc("POST /payments/{id}/process", handler.ProcessPayment)


	addr := ":8080"
	log.Printf("payments service listening on %s", addr)
	if err := http.ListenAndServe(addr, loggingMiddleware(mux)); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	status := "ok"
	dbStatus := "ok"
	code := http.StatusOK

	if err := db.PingContext(ctx); err != nil {
		dbStatus = "unreachable: " + err.Error()
		status = "degraded"
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{
		"status":   status,
		"database": dbStatus,
	})
}


func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// call the next handler in the chain
		next.ServeHTTP(w, r)

		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
		
	})

	
}
	