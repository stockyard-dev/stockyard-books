package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/stockyard-dev/stockyard-books/internal/server"
	"github.com/stockyard-dev/stockyard-books/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9200"
	}
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./books-data"
	}

	db, err := store.Open(dataDir)
	if err != nil {
		log.Fatalf("books: open database: %v", err)
	}
	defer db.Close()

	srv := server.New(db)

	fmt.Printf("\n  Books — Self-hosted business bookkeeping\n")
	fmt.Printf("  ─────────────────────────────────\n")
	fmt.Printf("  Dashboard:  http://localhost:%s/ui\n", port)
	fmt.Printf("  API:        http://localhost:%s/api\n", port)
	fmt.Printf("  Data:       %s\n", dataDir)
	fmt.Printf("  ─────────────────────────────────\n\n")

	log.Printf("books: listening on :%s", port)
	if err := http.ListenAndServe(":"+port, srv); err != nil {
		log.Fatalf("books: %v", err)
	}
}
