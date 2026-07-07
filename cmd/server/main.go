package main

import (
	"log"
	"net/http"
	"os"

	"go-invoice-lite/internal/app"
	"go-invoice-lite/internal/auth"
	"go-invoice-lite/internal/store"
)

func main() {
	addr := getenv("APP_ADDR", ":8080")
	dataPath := getenv("APP_DATA", "data/app.json")

	st, err := store.New(dataPath)
	if err != nil {
		log.Fatalf("store init failed: %v", err)
	}

	if st.UserCount() == 0 {
		hash, err := auth.HashPassword("admin123")
		if err != nil {
			log.Fatalf("seed password failed: %v", err)
		}
		if _, err := st.CreateUser("admin", hash, "admin"); err != nil {
			log.Fatalf("seed admin failed: %v", err)
		}
		log.Println("seeded default admin user: admin / admin123")
	}

	server := &http.Server{
		Addr:    addr,
		Handler: app.New(st),
	}

	log.Printf("GoInvoice Lite listening on http://localhost%s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
