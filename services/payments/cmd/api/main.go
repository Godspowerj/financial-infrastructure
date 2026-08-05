package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
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
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/webhooks/bank", handler.HandleBankWebhook)
	mux.HandleFunc("/payments", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/payments" || r.URL.Path == "/payments/" {
			handler.CreatePayment(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/process") {
			handler.ProcessPayment(w, r)
			return
		}
		handler.GetPayment(w, r)
	})
	mux.HandleFunc("/payments/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/process") {
			handler.ProcessPayment(w, r)
			return
		}
		handler.GetPayment(w, r)
	})

	addr := ":8080"
	log.Printf("payments service listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
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
