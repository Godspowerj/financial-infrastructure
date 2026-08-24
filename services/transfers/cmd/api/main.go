package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	transfers "fintech-lab/services/transfers/internal"
)

// db is the package-level connection pool, accessible by healthHandler.
var db *sql.DB

func main() {
	// --- Read Configuration ---
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://fintech:fintech@localhost:5432/transfers?sslmode=disable"
	}

	addr := os.Getenv("ADDRESS")
	if addr == "" {
		addr = ":8083"
	}

	// --- Connect to Postgres ---
	var err error
	db, err = sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("failed to configure database: %v", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	ledgerURL := os.Getenv("LEDGER_SERVICE_URL")
	if ledgerURL == "" {
		ledgerURL = "http://localhost:8082"
	}

	// --- Wire Up the 3 Layers (Dependency Injection) ---

	repo := transfers.NewRepository(db)
	ledgerClient := transfers.NewHTTPClient(ledgerURL)
	service := transfers.NewService(repo, ledgerClient)
	handler := transfers.NewHandler(service)

	// --- Register Routes ---
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)

	// Register transfer endpoints
	mux.HandleFunc("POST /transfers", handler.CreateTransfer)
	mux.HandleFunc("GET /transfers/{id}", handler.GetTransfer)
	// mux.HandleFunc("POST /transfers/{id}/process", handler.ProcessTransfer)

	// --- Start Server ---
	log.Printf("Starting transfers service on %s\n", addr)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

// healthHandler checks if the process is alive and can reach Postgres.
// Same pattern used in Payments (:8080) and Ledger (:8082).
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
