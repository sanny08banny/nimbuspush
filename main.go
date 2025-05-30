package main

import (
	"fmt"
	"log"
	"net/http"
	"nimbuspush/config"
	api "nimbuspush/internal/push"
	"nimbuspush/internal/transport"

	"github.com/gorilla/mux"
)

func main() {
	// Connect to database
	config.ConnectDatabase()


	router := mux.NewRouter()
router.HandleFunc("/register", api.RegisterDevice).Methods("POST")
router.HandleFunc("/push", api.PushMessage).Methods("POST") // ✅ Add this line

	go func() {
		fmt.Println("Starting HTTP REST API on :8080")
		if err := http.ListenAndServe(":8080", router); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Start QUIC server on :4242 (blocking)
	if err := transport.StartServer(":4242"); err != nil {
		log.Fatalf("QUIC server failed: %v", err)
	}

	select{}
}

