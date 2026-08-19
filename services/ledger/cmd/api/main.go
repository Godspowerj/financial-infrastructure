package main

import (
	"database/sql"
	"log"
	"os"
	"time"
	"net/http"
	"context"
	"encoding/json"

	_ "github.com/jackc/pgx/v5/stdlib"

	ledger "fintech-lab/services/ledger/internal"

)

var db *sql.DB

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://fintech:fintech@localhost:5432/ledger?sslmode=disable"
	}

	var err error
	db, err = sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("failed to configure database: %v", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	repo := ledger.NewRepository(db)
    service := ledger.NewService(repo)
    handler := ledger.NewHandler(service)
	

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
    mux.HandleFunc("POST /ledger/entries", handler.CreateEntryWithLines)
	mux.HandleFunc("GET /ledger", handler.GetLedgerEntries)
	//mux.HandleFunc("GET /ledger/{id}", handler.GetLedgerEntryByID)


    addr := ":8082"
	log.Printf("Staring ledger service on %s\n", addr)

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




