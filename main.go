package main

import (
	"fmt"
	"log"
	"net/http"
	"nimbuspush/config"
	api "nimbuspush/internal/push"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

func main() {
    // Init DB
    config.ConnectDatabase()


    // Init Kafka producer
    producer, err := kafka.NewProducer(&kafka.ConfigMap{
        "bootstrap.servers": "localhost:9092",
    })
    if err != nil {
        log.Fatalf("Kafka producer init failed: %v", err)
    }
    defer producer.Close()

    api.Producer = producer
    api.KafkaTopic = "nimbus-messages"

    // Routes
    http.HandleFunc("/register", api.RegisterDevice)
    http.HandleFunc("/publish", api.PublishMessage)

    fmt.Println("Server running on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}


